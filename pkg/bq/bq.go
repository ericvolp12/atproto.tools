package bq

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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

	tableLk   sync.RWMutex
	tableDate string
	inserter  *bigquery.Inserter
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

	return &BQ{
		recordSchema: recordSchema,
		client:       bqClient,
		dataset:      bqDataset,
		logger:       logger,
		tablePrefix:  tablePrefix,
	}, nil
}

func (bq *BQ) InsertRecord(ctx context.Context, record *Record) error {
	ctx, span := tracer.Start(ctx, "InsertRecord")
	defer span.End()

	if err := bq.CreateTableIfNotExists(ctx); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	span.SetAttributes(
		attribute.String("repo", record.Repo),
		attribute.String("collection", record.Collection),
		attribute.String("r_key", record.RKey),
		attribute.String("action", record.Action),
		attribute.Int64("firehose_seq", record.FirehoseSeq),
	)

	bq.tableLk.RLock()
	defer bq.tableLk.RUnlock()
	return bq.inserter.Put(ctx, record)
}

func (bq *BQ) CreateTableIfNotExists(ctx context.Context) error {
	today := time.Now().Format("20060102")

	bq.tableLk.RLock()

	if bq.tableDate == today && bq.inserter != nil {
		bq.tableLk.RUnlock()
		return nil
	}
	bq.tableLk.RUnlock()

	bq.tableLk.Lock()
	defer bq.tableLk.Unlock()

	table := bq.dataset.Table(fmt.Sprintf("%s_%s", bq.tablePrefix, today))
	_, err := table.Metadata(ctx)
	if err != nil {
		bq.logger.Info("table does not exist, creating", "table", table.FullyQualifiedName())
		if err := table.Create(ctx, &bigquery.TableMetadata{Schema: bq.recordSchema}); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

func (bq *BQ) Close() error {
	return bq.client.Close()
}
