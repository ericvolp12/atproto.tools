package parq

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"

	"github.com/parquet-go/parquet-go"
)

type Record struct {
	CreatedAt   int64  `parquet:"created_at"`
	FirehoseSeq int64  `parquet:"firehose_seq"`
	Repo        string `parquet:"repo"`
	Collection  string `parquet:"collection"`
	RKey        string `parquet:"r_key"`
	Action      string `parquet:"action"`
	Raw         string `parquet:"raw"`
	Error       string `parquet:"error"`
}

type Parq struct {
	logger       *slog.Logger
	fileDir      string
	prefix       string
	writeQueue   chan *Record
	shutdown     chan struct{}
	wg           sync.WaitGroup
	batchSize    int
	maxBatchWait time.Duration
}

func NewParq(logger *slog.Logger, fileDir, prefix string, batchSize int, maxBatchWait time.Duration) (*Parq, error) {
	p := Parq{
		logger:       logger,
		fileDir:      fileDir,
		prefix:       prefix,
		batchSize:    batchSize,
		maxBatchWait: maxBatchWait,
		writeQueue:   make(chan *Record, batchSize*2),
		shutdown:     make(chan struct{}),
	}

	// Make sure the file directory exists
	err := os.MkdirAll(fileDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create parquet file directory: %w", err)
	}

	return &p, nil
}

// StartWriter starts the writer goroutine which writes records to parquet files
// when the batch size is reached, after every maxBatchWait duration, or when the shutdown signal is received
func (p *Parq) StartWriter() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		var records []*Record
		t := time.NewTicker(p.maxBatchWait)
		defer t.Stop()

		p.logger.Info("starting parquet writer loop")

		for {
			select {
			case r := <-p.writeQueue:
				records = append(records, r)
				if len(records) >= p.batchSize {
					p.logger.Info("writing parquet file due to max batch size", "batch_size", p.batchSize)
					err := p.WriteFile(records)
					if err != nil {
						p.logger.Error("failed to write parquet file", "error", err)
					}
					records = nil
				}
			case <-t.C:
				p.logger.Info("writing parquet file due to max batch wait", "max_batch_wait", p.maxBatchWait.String())
				if len(records) > 0 {
					err := p.WriteFile(records)
					if err != nil {
						p.logger.Error("failed to write parquet file", "error", err)
					}
					records = nil
				}
			case <-p.shutdown:
				p.logger.Info("shutting down parquet writer")
				if len(records) > 0 {
					err := p.WriteFile(records)
					if err != nil {
						p.logger.Error("failed to write parquet file", "error", err)
					}
				}
				return
			}
		}
	}()
}

// Shutdown signals the writer goroutine to shutdown
func (p *Parq) Shutdown() {
	p.logger.Info("waiting for parquet writer to shutdown")
	close(p.shutdown)
	p.wg.Wait()
	p.logger.Info("parquet writer shutdown successfully")
}

// EnqueueRecords enqueues the given records to be written to a parquet file
func (p *Parq) EnqueueRecords(records []*Record) {
	for _, r := range records {
		p.writeQueue <- r
	}
}

// WriteFile writes the given records to a parquet file
func (p *Parq) WriteFile(records []*Record) error {
	// Write files to a parquet file with the current timestamp as the file suffix
	fName := path.Join(p.fileDir, fmt.Sprintf("%s_%s.parquet", p.prefix, time.Now().UTC().Format("2006_01_02-15_04_05")))

	filterBits := uint(10)

	p.logger.Info("writing parquet file", "file_path", fName, "num_records", len(records))

	err := parquet.WriteFile(fName, records, parquet.BloomFilters(
		parquet.SplitBlockFilter(filterBits, "repo"),
		parquet.SplitBlockFilter(filterBits, "collection"),
		parquet.SplitBlockFilter(filterBits, "r_key"),
		parquet.SplitBlockFilter(filterBits, "action"),
	))
	if err != nil {
		return fmt.Errorf("failed to write parquet file: %w", err)
	}

	p.logger.Info("wrote parquet file", "file_path", fName)

	return nil
}
