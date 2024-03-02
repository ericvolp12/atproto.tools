package bq

import (
	"time"

	"cloud.google.com/go/bigquery"
	"gorm.io/gorm"
)

type Cursor struct {
	gorm.Model
	LastSeq int64
}

type Record struct {
	CreatedAt time.Time `bigquery:"created_at"`

	FirehoseSeq int64             `bigquery:"firehose_seq"`
	Repo        string            `bigquery:"repo"`
	Collection  string            `bigquery:"collection"`
	RKey        string            `bigquery:"r_key"`
	Action      string            `bigquery:"action"`
	Raw         bigquery.NullJSON `bigquery:"raw"`

	Error string `bigquery:"error"`
}
