package bq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/bigquery"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type BQ struct {
	logger       *slog.Logger
	recordSchema bigquery.Schema
	client       *bigquery.Client
	dataset      *bigquery.Dataset

	tablePrefix string

	tableDate string
	inserter  *bigquery.Inserter

	recordBuf chan *Record
}

var tracer = otel.Tracer("bq")

func NewBQ(
	ctx context.Context,
	projectID string,
	dataset string,
	tablePrefix string,
	logger *slog.Logger,
) (*BQ, error) {
	recordSchema, err := bigquery.InferSchema(Record{})
	if err != nil {
		return nil, fmt.Errorf("failed to infer schema: %w", err)
	}

	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bigquery client: %w", err)
	}

	bqDataset := bqClient.Dataset(dataset)

	if _, err := bqDataset.Metadata(ctx); err != nil {
		return nil, fmt.Errorf("failed to get dataset metadata, make sure to create it if it doesn't exist: %w", err)
	}

	bq := &BQ{
		recordSchema: recordSchema,
		client:       bqClient,
		dataset:      bqDataset,
		logger:       logger,
		tablePrefix:  tablePrefix,
		recordBuf:    make(chan *Record, 100_000),
	}

	// Start a routine to batch insert records every 5 seconds
	go func() {
		t := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-t.C:
				if err := bq.insertRecords(ctx); err != nil {
					logger.Error("failed to insert records", "error", err)
				}
			}
		}
	}()

	return bq, nil
}

func (bq *BQ) InsertRecord(ctx context.Context, record *Record) error {
	ctx, span := tracer.Start(ctx, "InsertRecord")
	defer span.End()

	span.SetAttributes(
		attribute.String("repo", record.Repo),
		attribute.String("collection", record.Collection),
		attribute.String("r_key", record.RKey),
		attribute.String("action", record.Action),
		attribute.Int64("firehose_seq", record.FirehoseSeq),
	)

	bq.recordBuf <- record

	recordsProcessed.WithLabelValues(bq.tablePrefix).Inc()
	queueDepth.WithLabelValues(bq.tablePrefix).Inc()

	return nil
}

func (bq *BQ) insertRecords(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "insertRecords")
	defer span.End()

	// Create table if it doesn't exist
	if err := bq.CreateTableIfNotExists(ctx); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Grab up to 10_000 records from the buffer
	batchSize := 10_000

	records := make([]*Record, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		select {
		case record := <-bq.recordBuf:
			records = append(records, record)
			queueDepth.WithLabelValues(bq.tablePrefix).Dec()
		default:
			break
		}
	}

	// If there are no records, return early
	if len(records) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		batchSubmissionDuration.WithLabelValues(bq.tablePrefix).Observe(float64(elapsed.Milliseconds()))
		batchSizeHist.WithLabelValues(bq.tablePrefix).Observe(float64(batchSize))
	}()

	// Insert the records
	if err := bq.inserter.Put(ctx, records); err != nil {
		return fmt.Errorf("failed to insert records: %w", err)
	}

	return nil
}

func (bq *BQ) CreateTableIfNotExists(ctx context.Context) error {
	today := time.Now().Format("20060102")

	if bq.tableDate == today && bq.inserter != nil {
		return nil
	}

	table := bq.dataset.Table(fmt.Sprintf("%s_%s", bq.tablePrefix, today))
	_, err := table.Metadata(ctx)
	if err != nil {
		bq.logger.Info("table does not exist, creating", "table", table.FullyQualifiedName())
		if err := table.Create(ctx, &bigquery.TableMetadata{Schema: bq.recordSchema}); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	bq.inserter = table.Inserter()

	return nil
}

func (bq *BQ) Close() error {
	return bq.client.Close()
}
