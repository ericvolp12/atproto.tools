package bq

import (
	"context"
	"fmt"
	"log/slog"

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

	span.SetAttributes(
		attribute.String("repo", record.Repo),
		attribute.String("collection", record.Collection),
		attribute.String("r_key", record.RKey),
		attribute.String("action", record.Action),
		attribute.Int64("firehose_seq", record.FirehoseSeq),
	)

	// Date-based table partitioning
	tableName := fmt.Sprintf("%s_%s", bq.tablePrefix, record.CreatedAt.Format("20060102"))
	table := bq.dataset.Table(tableName)

	u := table.Inserter()

	return u.Put(ctx, record)
}

func (bq *BQ) Close() error {
	return bq.client.Close()
}
