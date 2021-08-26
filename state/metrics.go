package state

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	// MetricsSubsystem is a subsystem shared by all metrics exposed by this
	// package.
	MetricsSubsystem = "state"
)

// Metrics contains metrics exposed by this package.
type Metrics struct {
	// Time between BeginBlock and EndBlock.
	BlockProcessingTime metrics.Histogram
	// Time gauge between BeginBlock and EndBlock.
	BlockExecutionTime metrics.Gauge
	// Time of commit
	BlockCommitTime metrics.Gauge
	// Time of app commit
	BlockAppCommitTime metrics.Gauge
	// Time of update mempool
	BlockUpdateMempoolTime metrics.Gauge
}

// PrometheusMetrics returns Metrics build using Prometheus client library.
// Optionally, labels can be provided along with their values ("foo",
// "fooValue").
func PrometheusMetrics(namespace string, labelsAndValues ...string) *Metrics {
	labels := []string{}
	for i := 0; i < len(labelsAndValues); i += 2 {
		labels = append(labels, labelsAndValues[i])
	}
	compositeBuckets := stdprometheus.LinearBuckets(20, 20, 5)
	compositeBuckets = append(compositeBuckets, stdprometheus.LinearBuckets(200, 100, 4)...)
	compositeBuckets = append(compositeBuckets, stdprometheus.LinearBuckets(1000, 500, 4)...)

	return &Metrics{
		BlockProcessingTime: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_processing_time",
			Help:      "Time between BeginBlock and EndBlock in ms.",
			Buckets:   compositeBuckets,
		}, labels).With(labelsAndValues...),
		BlockExecutionTime: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_execution_time",
			Help:      "Time between BeginBlock and EndBlock in ms.",
		}, labels).With(labelsAndValues...),
		BlockCommitTime: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_commit_time",
			Help:      "Time of commit in ms.",
		}, labels).With(labelsAndValues...),
		BlockAppCommitTime: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_app_commit_time",
			Help:      "Time of app commit in ms.",
		}, labels).With(labelsAndValues...),
		BlockUpdateMempoolTime: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_update_mempool_time",
			Help:      "Time of update mempool in ms.",
		}, labels).With(labelsAndValues...),
	}
}

// NopMetrics returns no-op Metrics.
func NopMetrics() *Metrics {
	return &Metrics{
		BlockProcessingTime:    discard.NewHistogram(),
		BlockExecutionTime:     discard.NewGauge(),
		BlockCommitTime:        discard.NewGauge(),
		BlockAppCommitTime:     discard.NewGauge(),
		BlockUpdateMempoolTime: discard.NewGauge(),
	}
}
