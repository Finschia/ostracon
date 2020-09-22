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
	// Time of ValidBlock
	BlockVerifyingTime metrics.Histogram
	// Time between BeginBlock and EndBlock.
	BlockProcessingTime metrics.Histogram
	// Time of Commit
	BlockCommitingTime metrics.Histogram
}

// PrometheusMetrics returns Metrics build using Prometheus client library.
// Optionally, labels can be provided along with their values ("foo",
// "fooValue").
func PrometheusMetrics(namespace string, labelsAndValues ...string) *Metrics {
	labels := []string{}
	for i := 0; i < len(labelsAndValues); i += 2 {
		labels = append(labels, labelsAndValues[i])
	}
	compositeBuckets := stdprometheus.LinearBuckets(1, 20, 5)
	compositeBuckets = append(compositeBuckets, stdprometheus.LinearBuckets(101, 100, 4)...)
	compositeBuckets = append(compositeBuckets, stdprometheus.LinearBuckets(501, 500, 4)...)

	return &Metrics{
		BlockVerifyingTime: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_verifying_time",
			Help:      "Time of ValidBlock in ms.",
			Buckets:   stdprometheus.LinearBuckets(1, 50, 10),
		}, labels).With(labelsAndValues...),
		BlockProcessingTime: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_processing_time",
			Help:      "Time between BeginBlock and EndBlock in ms.",
			Buckets:   compositeBuckets,
		}, labels).With(labelsAndValues...),
		BlockCommitingTime: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_commiting_time",
			Help:      "Time of Commit in ms.",
			Buckets:   stdprometheus.LinearBuckets(1, 20, 10),
		}, labels).With(labelsAndValues...),
	}
}

// NopMetrics returns no-op Metrics.
func NopMetrics() *Metrics {
	return &Metrics{
		BlockVerifyingTime:  discard.NewHistogram(),
		BlockProcessingTime: discard.NewHistogram(),
		BlockCommitingTime:  discard.NewHistogram(),
	}
}
