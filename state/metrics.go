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
	// Time between BeginBlock and EndBlock.
	BlockExecutionTime metrics.Gauge
	// Time of Commit
	BlockCommitTime metrics.Gauge
	// Time of App Commit
	BlockAppCommitTime metrics.Gauge
}

// PrometheusMetrics returns Metrics build using Prometheus client library.
// Optionally, labels can be provided along with their values ("foo",
// "fooValue").
func PrometheusMetrics(namespace string, labelsAndValues ...string) *Metrics {
	labels := []string{}
	for i := 0; i < len(labelsAndValues); i += 2 {
		labels = append(labels, labelsAndValues[i])
	}
	return &Metrics{
		BlockProcessingTime: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_processing_time",
			Help:      "Time between BeginBlock and EndBlock in ms.",
			Buckets:   stdprometheus.LinearBuckets(1, 10, 10),
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
			Help:      "Time of Commit in ms.",
		}, labels).With(labelsAndValues...),
		BlockAppCommitTime: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_app_commit_time",
			Help:      "Time of App Commit in ms.",
		}, labels).With(labelsAndValues...),
	}
}

// NopMetrics returns no-op Metrics.
func NopMetrics() *Metrics {
	return &Metrics{
		BlockProcessingTime: discard.NewHistogram(),
		BlockExecutionTime:  discard.NewGauge(),
		BlockCommitTime:     discard.NewGauge(),
		BlockAppCommitTime:  discard.NewGauge(),
	}
}
