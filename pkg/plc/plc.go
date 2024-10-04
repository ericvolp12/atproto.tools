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
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/time/rate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	Writer        *gorm.DB
	Reader        *gorm.DB
	Limiter       *rate.Limiter

	Client   *http.Client
	shutdown chan chan error
}

var tracer = otel.Tracer("plc")

func NewPLC(ctx context.Context, host, dataDir string, logger *slog.Logger, checkInterval time.Duration) (*PLC, error) {
	logger = logger.With("module", "plc")

	// Initialize a SQLite database
	writerDB, err := gorm.Open(sqlite.Open(filepath.Join(dataDir, "plc.db")), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Migrate the database schema
	err = writerDB.AutoMigrate(&Cursor{}, &DBOp{}, &DBDid{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Set Pragmas
	err = writerDB.Exec("PRAGMA journal_mode=WAL;").Error
	if err != nil {
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}

	err = writerDB.Exec("PRAGMA synchronous=normal;").Error
	if err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	writerRawDB, err := writerDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw database: %w", err)
	}

	// Set Max Open Connections
	writerRawDB.SetMaxOpenConns(1)

	// Initialize the Reader database
	readerDB, err := gorm.Open(sqlite.Open(filepath.Join(dataDir, "plc.db")), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open reader database: %w", err)
	}

	// Set Pragmas
	err = readerDB.Exec("PRAGMA journal_mode=WAL;").Error
	if err != nil {
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}

	err = readerDB.Exec("PRAGMA synchronous=normal;").Error
	if err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	readerRawDB, err := readerDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw reader database: %w", err)
	}

	// Set Max Open Connections
	readerRawDB.SetMaxOpenConns(50)

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	limiter := rate.NewLimiter(rate.Limit(1), 1)

	cursor := &Cursor{}
	err = writerDB.First(cursor).Error
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
		Writer:        writerDB,
		Reader:        readerDB,
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
	dbDids := make([]*DBDid, 0)

	// Response is JSONLines
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		var op PLCOp
		err := dec.Decode(&op)
		if err != nil {
			return 0, fmt.Errorf("failed to decode JSON: %w", err)
		}

		dbOp, err := op.ToDBOp()
		if err != nil {
			return 0, fmt.Errorf("failed to convert op to dbOp: %w", err)
		}

		dbDids = append(dbDids, &DBDid{DID: op.DID, CreatedAt: op.CreatedAt})
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

	err = plc.Writer.CreateInBatches(dbOps, 100).Error
	if err != nil {
		return 0, fmt.Errorf("failed to save ops: %w", err)
	}

	err = plc.Writer.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(dbDids, 100).Error
	if err != nil {
		return 0, fmt.Errorf("failed to save dids: %w", err)
	}

	err = plc.Writer.Save(plc.Cursor).Error
	if err != nil {
		return 0, fmt.Errorf("failed to save cursor: %w", err)
	}

	return newOps, nil
}

type DIDDocument struct {
	Context []string `json:"@context"`
	identity.DIDDocument
}

type DBOp struct {
	gorm.Model
	DID       string    `gorm:"index:idx_did_cid;index:idx_did_created_at"`
	CID       string    `gorm:"index:idx_did_cid"`
	CreatedAt time.Time `gorm:"index:idx_did_created_at,sort:desc"`
	Nullified bool
	Operation []byte
	PDS       string `gorm:"index:idx_pds"`
	Handle    string `gorm:"index:idx_handle"`
}

type DBDid struct {
	Num       uint64 `gorm:"primarykey"`
	DID       string `gorm:"uniqueIndex"`
	CreatedAt time.Time
}

type PLCOp struct {
	DID       string    `json:"did"`
	CID       string    `json:"cid"`
	CreatedAt time.Time `json:"createdAt"`
	Nullified bool      `json:"nullified"`
	Operation any       `json:"operation"`
}

var contexts = []string{
	"https://www.w3.org/ns/did/v1",
	"https://w3id.org/security/multikey/v1",
	"https://w3id.org/security/suites/secp256k1-2019/v1",
}

var ErrNotFound = errors.New("not found")

func (plc *PLC) GetDIDDocument(ctx context.Context, did string) (*DIDDocument, error) {
	ctx, span := tracer.Start(ctx, "GetDIDDocument")
	defer span.End()

	// Get the latest DB op for the DID and unpack the operation into an identity.DIDDocument
	var dbOp DBOp
	err := plc.Reader.Where("d_id = ?", did).Order("created_at desc").First(&dbOp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get latest op: %w", err)
	}

	op, err := dbOp.ToOp()

	doc := &DIDDocument{Context: contexts}
	doc.DID = syntax.DID(did)

	// Cast the operation to a map
	opMap, ok := op.Operation.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to cast operation to map")
	}

	// Unpack Handles/AKAs
	akaInt, ok := opMap["alsoKnownAs"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to cast alsoKnownAs to []interface{}")
	}

	aka := []string{}
	for _, akaRaw := range akaInt {
		akaStr, ok := akaRaw.(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast alsoKnownAs to string")
		}
		aka = append(aka, akaStr)
	}

	doc.AlsoKnownAs = aka

	// Unpack Services
	doc.Service = []identity.DocService{}
	opServices, ok := opMap["services"].(map[string]interface{})
	if ok {
		for id, svc := range opServices {
			svcMap, ok := svc.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("service is not a map")
			}

			svcType, ok := svcMap["type"].(string)
			if !ok {
				return nil, fmt.Errorf("service map does not contain a 'type' key")
			}

			svcEndpoint, ok := svcMap["endpoint"].(string)
			if !ok {
				return nil, fmt.Errorf("service map does not contain a 'endpoint' key")
			}

			doc.Service = append(doc.Service, identity.DocService{
				ID:              fmt.Sprintf("#%s", id),
				Type:            svcType,
				ServiceEndpoint: svcEndpoint,
			})
		}
	}

	// Unpack Verification Methods
	doc.VerificationMethod = []identity.DocVerificationMethod{}
	opVerificationMethods, ok := opMap["verificationMethods"].(map[string]interface{})
	if ok {
		for id, key := range opVerificationMethods {
			vmID := fmt.Sprintf("%s#%s", did, id)
			key, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("failed to cast verification method key to string")
			}
			// Trim the did:key: prefix
			key = strings.TrimPrefix(key, "did:key:")

			doc.VerificationMethod = append(doc.VerificationMethod, identity.DocVerificationMethod{
				ID:                 vmID,
				Type:               "Multikey",
				Controller:         did,
				PublicKeyMultibase: key,
			})
		}
	}

	return doc, nil
}

func (plc *PLC) GetDIDByHandle(ctx context.Context, handle string) (string, error) {
	ctx, span := tracer.Start(ctx, "GetDIDByHandle")
	defer span.End()

	var dbOp DBOp
	err := plc.Reader.Where("handle = ?", handle).Order("created_at desc").First(&dbOp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get latest op: %w", err)
	}

	return dbOp.DID, nil
}

func (plc *PLC) GetHandleByDID(ctx context.Context, did string) (string, error) {
	ctx, span := tracer.Start(ctx, "GetHandleByDID")
	defer span.End()

	var dbOp DBOp
	err := plc.Reader.Where("d_id = ?", did).Order("created_at desc").First(&dbOp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get latest op: %w", err)
	}

	return dbOp.Handle, nil
}

// GetSig returns the value of the "sig" key in the Operation map
func (op *PLCOp) GetSig() (string, error) {
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

func (op *PLCOp) ToDBOp() (*DBOp, error) {
	opJSON, err := json.Marshal(op.Operation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal op: %w", err)
	}

	// Extract "handle" from the operation "alsoKnownAs" array
	handle := ""
	opMap, ok := op.Operation.(map[string]interface{})
	if ok {
		alsoKnownAs, ok := opMap["alsoKnownAs"].([]interface{})
		if ok {
			for _, aka := range alsoKnownAs {
				handle, ok = aka.(string)
				if ok {
					// Trim the at:// prefix
					handle = strings.TrimPrefix(handle, "at://")
					break
				}
			}
		}
	}

	// Extract PDS from the operation "services.atproto_pds.endpoint" key
	pds := ""
	if ok {
		if services, ok := opMap["services"].(map[string]interface{}); ok {
			if atprotoPDS, ok := services["atproto_pds"].(map[string]interface{}); ok {
				if endpoint, ok := atprotoPDS["endpoint"].(string); ok {
					pds = endpoint
				}
			}
		}
	}

	return &DBOp{
		DID:       op.DID,
		CID:       op.CID,
		CreatedAt: op.CreatedAt,
		Nullified: op.Nullified,
		Operation: opJSON,
		Handle:    handle,
		PDS:       pds,
	}, nil
}

func (op *DBOp) ToOp() (*PLCOp, error) {
	var innerOp any
	err := json.Unmarshal(op.Operation, &innerOp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation: %w", err)
	}

	return &PLCOp{
		DID:       op.DID,
		CID:       op.CID,
		CreatedAt: op.CreatedAt,
		Nullified: op.Nullified,
		Operation: innerOp,
	}, nil
}
