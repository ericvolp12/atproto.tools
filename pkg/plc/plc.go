package plc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/time/rate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Cursor struct {
	gorm.Model
	DID           string
	CID           string
	LastCreatedAt time.Time
	OpsSeen       int
}

type PLC struct {
	Logger        *slog.Logger
	Host          string
	Cursor        *Cursor
	PageSize      int
	CheckInterval time.Duration
	DB            *gorm.DB
	Limiter       *rate.Limiter

	Client   *http.Client
	shutdown chan chan error
}

var tracer = otel.Tracer("plc")

func NewPLC(ctx context.Context, host, dataDir string, logger *slog.Logger, checkInterval time.Duration) (*PLC, error) {
	logger = logger.With("module", "plc")

	// Initialize a SQLite database
	db, err := gorm.Open(sqlite.Open(filepath.Join(dataDir, "plc.db")), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set Pragmas
	err = db.Exec("PRAGMA journal_mode=WAL;").Error
	if err != nil {
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}

	err = db.Exec("PRAGMA synchronous=normal;").Error
	if err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	// Migrate the database schema
	err = db.AutoMigrate(&Cursor{}, &DBOp{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	limiter := rate.NewLimiter(rate.Limit(1), 1)

	cursor := &Cursor{}
	err = db.First(cursor).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to get cursor: %w", err)
		}
	}

	return &PLC{
		Logger:        logger,
		Host:          host,
		PageSize:      1000,
		CheckInterval: checkInterval,
		DB:            db,
		Client:        client,
		Cursor:        cursor,
		Limiter:       limiter,
		shutdown:      make(chan chan error),
	}, nil
}

func (plc *PLC) Shutdown(ctx context.Context) error {
	plc.Logger.Info("attempting to shutdown PLC")
	errCh := make(chan error)
	plc.shutdown <- errCh
	return <-errCh
}

func (plc *PLC) Run(ctx context.Context) error {
	plc.Logger.Info("running")

	var opsSeen int
	var err error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case errCh := <-plc.shutdown:
			plc.Logger.Info("shutting down run loop")
			errCh <- nil
			return nil
		default:
		}

		opsSeen, err = plc.GetNextPage(ctx)
		if err != nil {
			plc.Logger.Error("failed to get next page", "err", err)
			if err == ErrRateLimited {
				plc.Logger.Info("rate limited, waiting 2 minutes")
				<-time.After(2 * time.Minute)
			} else {
				plc.Logger.Info("waiting 5 seconds before retrying")
				<-time.After(5 * time.Second)
			}
			continue
		}

		plc.Logger.Info("got next page", "opsSeen", opsSeen)

		if opsSeen < plc.PageSize {
			<-time.After(plc.CheckInterval)
		}
	}
}

var ErrRateLimited = errors.New("rate limited")

func (plc *PLC) GetNextPage(ctx context.Context) (int, error) {
	ctx, span := tracer.Start(ctx, "GetNextPage")
	defer span.End()

	after := ""
	if plc.Cursor.ID != 0 {
		after = fmt.Sprintf("&after=%s", plc.Cursor.LastCreatedAt.Format(time.RFC3339Nano))
	}

	u, err := url.Parse(fmt.Sprintf("%s/export?count=%d%s", plc.Host, plc.PageSize, after))
	if err != nil {
		return 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	plc.Logger.Info("getting next page", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "jaz-plc-mirror")

	// Rate limit requests
	err = plc.Limiter.Wait(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to wait for rate limiter: %w", err)
	}
	resp, err := plc.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			plc.Logger.Warn("rate limited")
			return 0, ErrRateLimited
		}
		return 0, fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	newOps := 0

	dbOps := make([]*DBOp, 0)

	// Response is JSONLines
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		var op Op
		err := dec.Decode(&op)
		if err != nil {
			return 0, fmt.Errorf("failed to decode JSON: %w", err)
		}

		dbOp, err := op.ToDBOp()
		if err != nil {
			return 0, fmt.Errorf("failed to convert op to dbOp: %w", err)
		}

		dbOps = append(dbOps, dbOp)

		newOps++
		plc.Cursor.DID = op.DID
		plc.Cursor.CID = op.CID
		plc.Cursor.LastCreatedAt = op.CreatedAt
		plc.Cursor.OpsSeen++
	}

	if len(dbOps) == 0 {
		return 0, nil
	}

	err = plc.DB.CreateInBatches(dbOps, 100).Error
	if err != nil {
		return 0, fmt.Errorf("failed to save ops: %w", err)
	}

	err = plc.DB.Save(plc.Cursor).Error
	if err != nil {
		return 0, fmt.Errorf("failed to save cursor: %w", err)
	}

	return newOps, nil
}

type DBOp struct {
	gorm.Model
	DID       string    `gorm:"index:idx_did_cid;index:idx_did_created_at"`
	CID       string    `gorm:"index:idx_did_cid"`
	CreatedAt time.Time `gorm:"index:idx_did_created_at,sort:desc"`
	Nullified bool
	Operation []byte
}

type Op struct {
	DID       string    `json:"did"`
	CID       string    `json:"cid"`
	CreatedAt time.Time `json:"createdAt"`
	Nullified bool      `json:"nullified"`
	Operation any       `json:"operation"`
}

// GetSig returns the value of the "sig" key in the Operation map
func (op *Op) GetSig() (string, error) {
	// Check if op.Operation is a map and has a "sig" string key
	// If it does, return the value of the "sig" key
	opMap, ok := op.Operation.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("operation is not a map")
	}

	sig, ok := opMap["sig"].(string)
	if !ok {
		return "", fmt.Errorf("operation map does not contain a 'sig' key")
	}

	return sig, nil
}

func (op *Op) ToDBOp() (*DBOp, error) {
	opJSON, err := json.Marshal(op)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal op: %w", err)
	}

	return &DBOp{
		DID:       op.DID,
		CID:       op.CID,
		CreatedAt: op.CreatedAt,
		Nullified: op.Nullified,
		Operation: opJSON,
	}, nil
}

func (op *DBOp) ToOp() (*Op, error) {
	var opJSON Op
	err := json.Unmarshal(op.Operation, &opJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal op: %w", err)
	}

	return &opJSON, nil
}
