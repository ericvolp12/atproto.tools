package bq

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var queueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "bq_queue_depth",
	Help: "The current depth of the BQ record buffer",
}, []string{"table"})

var recordsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "bq_records_processed",
	Help: "The number of records processed",
}, []string{"table"})

var batchSubmissionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "bq_batch_submission_duration",
	Help:    "The duration of time it takes to submit a batch of records to BQ",
	Buckets: prometheus.DefBuckets,
}, []string{"table"})

var batchSizeHist = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "bq_batch_size",
	Help:    "The size of a batch of records submitted to BQ",
	Buckets: prometheus.ExponentialBuckets(1, 2, 20),
}, []string{"table"})
