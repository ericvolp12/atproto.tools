package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/araddon/dateparse"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/data"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/parallel"
	"github.com/bluesky-social/indigo/repo"
	"github.com/gorilla/websocket"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	slogGorm "github.com/orandin/slog-gorm"
)

type Stream struct {
	logger    *slog.Logger
	socketURL *url.URL

	scheduler events.Scheduler

	lastSeq int64
	seqLk   sync.RWMutex

	streamClosed chan struct{}

	db  *gorm.DB
	ttl time.Duration
}

var tracer = otel.Tracer("stream")

func NewStream(
	logger *slog.Logger,
	socketURL string,
	sqlitePath string,
	migrate bool,
	ttl time.Duration,
) (*Stream, error) {
	gormLogger := slogGorm.New()

	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if migrate {
		db.AutoMigrate(&Event{})
		db.AutoMigrate(&Record{})
		db.AutoMigrate(&Cursor{})
	}

	// Set pragmas for performance
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA synchronous=normal;")

	u, err := url.Parse(socketURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse socket url: %w", err)
	}

	return &Stream{
		logger:       logger,
		socketURL:    u,
		streamClosed: make(chan struct{}),
		db:           db,
		ttl:          ttl,
	}, nil
}

func (s *Stream) Start(ctx context.Context) error {
	// Load the cursor if it exists
	var c Cursor
	if err := s.db.First(&c).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c = Cursor{}
			err := s.db.Create(&c).Error
			if err != nil {
				return fmt.Errorf("failed to create cursor: %w", err)
			}
		}
	}

	// Start a routine to save the cursor every 60 seconds
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for {
			select {
			case <-s.streamClosed:
				s.seqLk.RLock()
				c.LastSeq = s.lastSeq
				s.seqLk.RUnlock()
				s.logger.Info("stream closed, saving cursor", "seq", c.LastSeq)
				if err := s.db.Save(&c).Error; err != nil {
					s.logger.Error("failed to save cursor", "err", err)
				}
				s.logger.Info("cursor saved")
				return
			case <-ticker.C:
				s.seqLk.RLock()
				c.LastSeq = s.lastSeq
				s.seqLk.RUnlock()
				s.logger.Info("saving cursor", "seq", c.LastSeq)
				if err := s.db.Save(&c).Error; err != nil {
					s.logger.Error("failed to save cursor", "err", err)
				}
			}
		}
	}()

	// Start a routine to delete old events and records every 5 minutes
	if s.ttl > 0 {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			for {
				select {
				case <-s.streamClosed:
					return
				case <-ticker.C:
					s.logger.Info("deleting old events and records")
					if err := s.db.Exec("DELETE FROM events WHERE created_at < ?", time.Now().Add(-s.ttl)).Error; err != nil {
						s.logger.Error("failed to delete old events", "err", err)
					}

					if err := s.db.Exec("DELETE FROM records WHERE created_at < ?", time.Now().Add(-s.ttl)).Error; err != nil {
						s.logger.Error("failed to delete old records", "err", err)
					}

					s.logger.Info("old events and records deleted")
				}
			}
		}()
	}

	socketURL := s.socketURL
	if c.LastSeq != 0 {
		q := socketURL.Query()
		q.Set("seq", fmt.Sprintf("%d", c.LastSeq))
		socketURL.RawQuery = q.Encode()
	}

	rsc := events.RepoStreamCallbacks{
		RepoCommit:    s.RepoCommit,
		RepoHandle:    s.RepoHandle,
		RepoIdentity:  s.RepoIdentity,
		RepoInfo:      s.RepoInfo,
		RepoMigrate:   s.RepoMigrate,
		RepoTombstone: s.RepoTombstone,
		LabelLabels:   s.LabelLabels,
		LabelInfo:     s.LabelInfo,
		Error:         s.Error,
	}

	d := websocket.DefaultDialer

	s.logger.Info("connecting to relay", "url", socketURL.String())

	con, _, err := d.Dial(socketURL.String(), http.Header{
		"User-Agent": []string{"atp-looking-glass/0.0.1"},
	})

	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	scheduler := parallel.NewScheduler(100, 10, con.RemoteAddr().String(), rsc.EventHandler)

	s.scheduler = scheduler

	if err := events.HandleRepoStream(ctx, con, scheduler); err != nil {
		s.logger.Error("repo stream failed", "err", err)
	}

	s.logger.Info("repo stream shut down")

	close(s.streamClosed)

	return nil
}

func (s *Stream) SetSeq(seq int64) {
	s.seqLk.Lock()
	defer s.seqLk.Unlock()
	s.lastSeq = seq
}

func (s *Stream) GetSeq() int64 {
	s.seqLk.RLock()
	defer s.seqLk.RUnlock()
	return s.lastSeq
}

func (s *Stream) RepoCommit(evt *atproto.SyncSubscribeRepos_Commit) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "RepoCommit")
	defer span.End()

	logger := s.logger.With("repo", evt.Repo, "seq", evt.Seq)

	span.SetAttributes(
		attribute.String("repo", evt.Repo),
		attribute.Int64("seq", evt.Seq),
	)

	s.SetSeq(evt.Seq)

	// Record metadata about the event
	e := &Event{
		FirehoseSeq: evt.Seq,
		Repo:        evt.Repo,
		EventType:   "commit",
	}

	defer func() {
		if err := s.db.Create(e).Error; err != nil {
			s.logger.Error("failed to create event", "err", err)
		}
	}()

	if evt.TooBig {
		s.logger.Warn("commit too big", "repo", evt.Repo, "seq", evt.Seq)
		e.Error = "commit too big"
		return nil
	}

	r, err := repo.ReadRepoFromCar(ctx, bytes.NewReader(evt.Blocks))
	if err != nil {
		s.logger.Error("failed to read event repo", "err", err)
		e.Error = fmt.Sprintf("failed to read event repo: %v", err)
		return nil
	}

	t, err := dateparse.ParseAny(evt.Time)
	if err != nil {
		s.logger.Error("failed to parse time", "err", err)
		e.Error = fmt.Sprintf("failed to parse time: %v", err)
		return nil
	}

	e.Time = t.UnixNano()

	for _, op := range evt.Ops {
		switch op.Action {
		case "create", "update":
			if op.Cid == nil {
				logger.Warn("op missing cid", "path", op.Path, "action", op.Action)
				e.Error += fmt.Sprintf("op missing cid (path: %q)", op.Path)
				continue
			}

			c := (cid.Cid)(*op.Cid)
			cid, rec, err := r.GetRecordBytes(ctx, op.Path)
			if err != nil {
				logger.Error("failed to get record bytes", "err", err)
				e.Error += fmt.Sprintf("failed to get record bytes (path: %q): %v", op.Path, err)
				continue
			}

			if c != cid {
				logger.Warn("cid mismatch", "from_event", c, "from_blocks", cid)
				e.Error += fmt.Sprintf("cid mismatch (path: %q): from_event %q, from_blocks %q", op.Path, c, cid)
				continue
			}

			if rec == nil {
				logger.Warn("record not found", "cid", c, "path", op.Path)
				e.Error += fmt.Sprintf("record not found (nil bytes loaded from event blocks) path: %q", op.Path)
				continue
			}

			asCbor, err := data.UnmarshalCBOR(*rec)
			if err != nil {
				logger.Error("failed to unmarshal record from CBOR", "err", err, "cid", c, "path", op.Path)
				e.Error += fmt.Sprintf("failed to unmarshal record from CBOR (path: %q): %v", op.Path, err)
				continue
			}

			recJSON, err := json.Marshal(asCbor)
			if err != nil {
				logger.Error("failed to marshal record to JSON", "err", err)
				e.Error += fmt.Sprintf("failed to marshal record to JSON (path: %q): %v", op.Path, err)
				continue
			}

			recRawURI := fmt.Sprintf("at://%s/%s", evt.Repo, op.Path)
			recURI, err := syntax.ParseATURI(recRawURI)
			if err != nil {
				logger.Error("failed to parse record uri", "err", err)
				e.Error += fmt.Sprintf("failed to parse record uri (path: %q): %v", op.Path, err)
				continue
			}

			dbRecord := &Record{
				FirehoseSeq: evt.Seq,
				Repo:        recURI.Authority().String(),
				Collection:  recURI.Collection().String(),
				RKey:        recURI.RecordKey().String(),
				Action:      op.Action,
				Raw:         recJSON,
			}

			if err := s.db.Create(dbRecord).Error; err != nil {
				logger.Error("failed to create db record", "err", err)
				e.Error += fmt.Sprintf("failed to create db record (path: %q): %v", op.Path, err)
				continue
			}
		case "delete":
			recRawURI := fmt.Sprintf("at://%s/%s", evt.Repo, op.Path)
			recURI, err := syntax.ParseATURI(recRawURI)
			if err != nil {
				logger.Error("failed to parse record uri", "err", err)
				e.Error += fmt.Sprintf("failed to parse record uri (path: %q): %v", op.Path, err)
				continue
			}

			dbRecord := &Record{
				FirehoseSeq: evt.Seq,
				Repo:        recURI.Authority().String(),
				Collection:  recURI.Collection().String(),
				RKey:        recURI.RecordKey().String(),
				Action:      op.Action,
			}

			if err := s.db.Create(dbRecord).Error; err != nil {
				logger.Error("failed to create db record", "err", err)
				e.Error += fmt.Sprintf("failed to create db record (path: %q): %v", op.Path, err)
				continue
			}
		default:
			logger.Warn("unknown action", "action", op.Action)
			e.Error += fmt.Sprintf("unknown action (path: %q): %q", op.Path, op.Action)
		}
	}

	return nil
}

func (s *Stream) RepoHandle(handle *atproto.SyncSubscribeRepos_Handle) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "RepoHandle")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("seq", handle.Seq),
	)

	s.SetSeq(handle.Seq)

	// Record metadata about the event
	e := &Event{
		FirehoseSeq: handle.Seq,
		Repo:        handle.Did,
		EventType:   "handle",
	}

	t, err := dateparse.ParseAny(handle.Time)
	if err != nil {
		s.logger.Error("failed to parse time", "err", err)
		e.Error = fmt.Sprintf("failed to parse time: %v", err)
		return nil
	}

	defer func() {
		if err := s.db.Create(e).Error; err != nil {
			s.logger.Error("failed to create event", "err", err)
		}
	}()

	e.Time = t.UnixNano()

	return nil

}

func (s *Stream) RepoIdentity(id *atproto.SyncSubscribeRepos_Identity) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "RepoIdentity")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("seq", id.Seq),
	)

	s.SetSeq(id.Seq)

	// Record metadata about the event
	e := &Event{
		FirehoseSeq: id.Seq,
		Repo:        id.Did,
		EventType:   "identity",
	}

	t, err := dateparse.ParseAny(id.Time)
	if err != nil {
		s.logger.Error("failed to parse time", "err", err)
		e.Error = fmt.Sprintf("failed to parse time: %v", err)
		return nil
	}

	defer func() {
		if err := s.db.Create(e).Error; err != nil {
			s.logger.Error("failed to create event", "err", err)
		}
	}()

	e.Time = t.UnixNano()

	return nil
}

func (s *Stream) RepoInfo(info *atproto.SyncSubscribeRepos_Info) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "RepoInfo")
	defer span.End()

	return nil
}

func (s *Stream) RepoMigrate(migrate *atproto.SyncSubscribeRepos_Migrate) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "RepoMigrate")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("seq", migrate.Seq),
	)

	s.SetSeq(migrate.Seq)

	// Record metadata about the event
	e := &Event{
		FirehoseSeq: migrate.Seq,
		Repo:        migrate.Did,
		EventType:   "migrate",
	}

	t, err := dateparse.ParseAny(migrate.Time)
	if err != nil {
		s.logger.Error("failed to parse time", "err", err)
		e.Error = fmt.Sprintf("failed to parse time: %v", err)
		return nil
	}

	defer func() {
		if err := s.db.Create(e).Error; err != nil {
			s.logger.Error("failed to create event", "err", err)
		}
	}()

	e.Time = t.UnixNano()

	return nil
}

func (s *Stream) RepoTombstone(tomb *atproto.SyncSubscribeRepos_Tombstone) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "RepoTombstone")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("seq", tomb.Seq),
	)

	s.SetSeq(tomb.Seq)

	// Record metadata about the event
	e := &Event{
		FirehoseSeq: tomb.Seq,
		Repo:        tomb.Did,
		EventType:   "tombstone",
	}

	t, err := dateparse.ParseAny(tomb.Time)
	if err != nil {
		s.logger.Error("failed to parse time", "err", err)
		e.Error = fmt.Sprintf("failed to parse time: %v", err)
		return nil
	}

	defer func() {
		if err := s.db.Create(e).Error; err != nil {
			s.logger.Error("failed to create event", "err", err)
		}
	}()

	e.Time = t.UnixNano()

	return nil
}

func (s *Stream) LabelLabels(label *atproto.LabelSubscribeLabels_Labels) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "LabelLabels")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("seq", label.Seq),
	)

	s.SetSeq(label.Seq)

	return nil
}

func (s *Stream) LabelInfo(info *atproto.LabelSubscribeLabels_Info) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "LabelInfo")
	defer span.End()

	return nil
}

func (s *Stream) Error(err *events.ErrorFrame) error {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "Error")
	defer span.End()

	return nil
}
