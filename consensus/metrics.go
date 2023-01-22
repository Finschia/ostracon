package consensus

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	// MetricsSubsystem is a subsystem shared by all metrics exposed by this
	// package.
	MetricsSubsystem = "consensus"
)

// Metrics contains metrics exposed by this package.
type Metrics struct {
	// Height of the chain.
	Height metrics.Gauge

	// ValidatorLastSignedHeight of a validator.
	ValidatorLastSignedHeight metrics.Gauge

	// Number of rounds.
	Rounds metrics.Gauge

	// Number of validators.
	Validators metrics.Gauge
	// Total power of all validators.
	ValidatorsPower metrics.Gauge
	// Power of a validator.
	ValidatorPower metrics.Gauge
	// Amount of blocks missed by a validator.
	ValidatorMissedBlocks metrics.Gauge
	// Number of validators who did not sign.
	MissingValidators metrics.Gauge
	// Total power of the missing validators.
	MissingValidatorsPower metrics.Gauge
	// Number of validators who tried to double sign.
	ByzantineValidators metrics.Gauge
	// Total power of the byzantine validators.
	ByzantineValidatorsPower metrics.Gauge

	// Time between this and the last block.
	BlockIntervalSeconds metrics.Gauge

	// Number of transactions.
	NumTxs metrics.Gauge
	// Size of the block.
	BlockSizeBytes metrics.Gauge
	// Total number of transactions.
	TotalTxs metrics.Gauge
	// The latest block height.
	CommittedHeight metrics.Gauge
	// Whether or not a node is fast syncing. 1 if yes, 0 if no.
	FastSyncing metrics.Gauge
	// Whether or not a node is state syncing. 1 if yes, 0 if no.
	StateSyncing metrics.Gauge

	// Number of blockparts transmitted by peer.
	BlockParts metrics.Counter

	// QuroumPrevoteMessageDelay is the interval in seconds between the proposal
	// timestamp and the timestamp of the earliest prevote that achieved a quorum
	// during the prevote step.
	//
	// To compute it, sum the voting power over each prevote received, in increasing
	// order of timestamp. The timestamp of the first prevote to increase the sum to
	// be above 2/3 of the total voting power of the network defines the endpoint
	// the endpoint of the interval. Subtract the proposal timestamp from this endpoint
	// to obtain the quorum delay.
	QuorumPrevoteMessageDelay metrics.Gauge

	// FullPrevoteMessageDelay is the interval in seconds between the proposal
	// timestamp and the timestamp of the latest prevote in a round where 100%
	// of the voting power on the network issued prevotes.
	FullPrevoteMessageDelay metrics.Gauge

	// ////////////////////////////////////
	// Metrics for measuring performance
	// ////////////////////////////////////

	// Number of blocks that are we couldn't receive
	MissingProposal metrics.Gauge

	// Number of rounds turned over.
	RoundFailures metrics.Histogram

	// Execution time profiling of each step
	DurationProposal           metrics.Histogram
	DurationPrevote            metrics.Histogram
	DurationPrecommit          metrics.Histogram
	DurationCommitExecuting    metrics.Histogram
	DurationCommitCommitting   metrics.Histogram
	DurationCommitRechecking   metrics.Histogram
	DurationWaitingForNewRound metrics.Histogram

	DurationGaugeProposal           metrics.Gauge
	DurationGaugePrevote            metrics.Gauge
	DurationGaugePrecommit          metrics.Gauge
	DurationGaugeCommitExecuting    metrics.Gauge
	DurationGaugeCommitCommitting   metrics.Gauge
	DurationGaugeCommitRechecking   metrics.Gauge
	DurationGaugeWaitingForNewRound metrics.Gauge
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
		Height: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "height",
			Help:      "Height of the chain.",
		}, labels).With(labelsAndValues...),
		Rounds: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "rounds",
			Help:      "Number of rounds.",
		}, labels).With(labelsAndValues...),

		Validators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "validators",
			Help:      "Number of validators.",
		}, labels).With(labelsAndValues...),
		ValidatorLastSignedHeight: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "validator_last_signed_height",
			Help:      "Last signed height for a validator",
		}, append(labels, "validator_address")).With(labelsAndValues...),
		ValidatorMissedBlocks: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "validator_missed_blocks",
			Help:      "Total missed blocks for a validator",
		}, append(labels, "validator_address")).With(labelsAndValues...),
		ValidatorsPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "validators_power",
			Help:      "Total power of all validators.",
		}, labels).With(labelsAndValues...),
		ValidatorPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "validator_power",
			Help:      "Power of a validator",
		}, append(labels, "validator_address")).With(labelsAndValues...),
		MissingValidators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "missing_validators",
			Help:      "Number of validators who did not sign.",
		}, labels).With(labelsAndValues...),
		MissingValidatorsPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "missing_validators_power",
			Help:      "Total power of the missing validators.",
		}, labels).With(labelsAndValues...),
		ByzantineValidators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "byzantine_validators",
			Help:      "Number of validators who tried to double sign.",
		}, labels).With(labelsAndValues...),
		ByzantineValidatorsPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "byzantine_validators_power",
			Help:      "Total power of the byzantine validators.",
		}, labels).With(labelsAndValues...),
		BlockIntervalSeconds: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_interval_seconds",
			Help:      "Time between this and the last block.",
		}, labels).With(labelsAndValues...),
		NumTxs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "num_txs",
			Help:      "Number of transactions.",
		}, labels).With(labelsAndValues...),
		BlockSizeBytes: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_size_bytes",
			Help:      "Size of the block.",
		}, labels).With(labelsAndValues...),
		TotalTxs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "total_txs",
			Help:      "Total number of transactions.",
		}, labels).With(labelsAndValues...),
		CommittedHeight: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "latest_block_height",
			Help:      "The latest block height.",
		}, labels).With(labelsAndValues...),
		FastSyncing: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "fast_syncing",
			Help:      "Whether or not a node is fast syncing. 1 if yes, 0 if no.",
		}, labels).With(labelsAndValues...),
		StateSyncing: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "state_syncing",
			Help:      "Whether or not a node is state syncing. 1 if yes, 0 if no.",
		}, labels).With(labelsAndValues...),
		BlockParts: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "block_parts",
			Help:      "Number of blockparts transmitted by peer.",
		}, append(labels, "peer_id")).With(labelsAndValues...),
		QuorumPrevoteMessageDelay: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "quorum_prevote_message_delay",
			Help: "Difference in seconds between the proposal timestamp and the timestamp " +
				"of the latest prevote that achieved a quorum in the prevote step.",
		}, labels).With(labelsAndValues...),
		FullPrevoteMessageDelay: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "full_prevote_message_delay",
			Help: "Difference in seconds between the proposal timestamp and the timestamp " +
				"of the latest prevote that achieved 100% of the voting power in the prevote step.",
		}, labels).With(labelsAndValues...),
		MissingProposal: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "missing_proposal",
			Help:      "Number of blocks we couldn't receive",
		}, labels).With(labelsAndValues...),
		RoundFailures: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "round_failures",
			Help:      "Number of rounds failed on consensus",
			Buckets:   stdprometheus.LinearBuckets(0, 1, 5),
		}, labels).With(labelsAndValues...),
		DurationProposal: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_proposal",
			Help:      "Duration of proposal step",
			Buckets:   stdprometheus.LinearBuckets(100, 100, 10),
		}, labels).With(labelsAndValues...),
		DurationPrevote: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_prevote",
			Help:      "Duration of prevote step",
			Buckets:   stdprometheus.LinearBuckets(100, 100, 10),
		}, labels).With(labelsAndValues...),
		DurationPrecommit: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_precommit",
			Help:      "Duration of precommit step",
			Buckets:   stdprometheus.LinearBuckets(100, 100, 10),
		}, labels).With(labelsAndValues...),
		DurationCommitExecuting: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_commit_executing",
			Help:      "Duration of executing block txs",
			Buckets:   stdprometheus.LinearBuckets(100, 100, 10),
		}, labels).With(labelsAndValues...),
		DurationCommitCommitting: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_commit_committing",
			Help:      "Duration of committing updated state",
			Buckets:   stdprometheus.LinearBuckets(100, 100, 10),
		}, labels).With(labelsAndValues...),
		DurationCommitRechecking: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_commit_rechecking",
			Help:      "Duration of rechecking mempool txs",
			Buckets:   stdprometheus.LinearBuckets(100, 100, 10),
		}, labels).With(labelsAndValues...),
		DurationWaitingForNewRound: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_waiting_for_new_round",
			Help:      "Duration of waiting for next new round",
			Buckets:   stdprometheus.LinearBuckets(100, 100, 10),
		}, labels).With(labelsAndValues...),
		DurationGaugeProposal: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_gauge_proposal",
			Help:      "Duration of proposal step",
		}, labels).With(labelsAndValues...),
		DurationGaugePrevote: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_gauge_prevote",
			Help:      "Duration of prevote step",
		}, labels).With(labelsAndValues...),
		DurationGaugePrecommit: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_gauge_precommit",
			Help:      "Duration of precommit step",
		}, labels).With(labelsAndValues...),
		DurationGaugeCommitExecuting: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_gauge_commit_executing",
			Help:      "Duration of executing block txs",
		}, labels).With(labelsAndValues...),
		DurationGaugeCommitCommitting: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_gauge_commit_committing",
			Help:      "Duration of committing updated state",
		}, labels).With(labelsAndValues...),
		DurationGaugeCommitRechecking: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_gauge_commit_rechecking",
			Help:      "Duration of rechecking mempool txs",
		}, labels).With(labelsAndValues...),
		DurationGaugeWaitingForNewRound: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_gauge_waiting_for_new_round",
			Help:      "Duration of waiting for next new round",
		}, labels).With(labelsAndValues...),
	}
}

// NopMetrics returns no-op Metrics.
func NopMetrics() *Metrics {
	return &Metrics{
		Height: discard.NewGauge(),

		ValidatorLastSignedHeight: discard.NewGauge(),

		Rounds: discard.NewGauge(),

		Validators:               discard.NewGauge(),
		ValidatorsPower:          discard.NewGauge(),
		ValidatorPower:           discard.NewGauge(),
		ValidatorMissedBlocks:    discard.NewGauge(),
		MissingValidators:        discard.NewGauge(),
		MissingValidatorsPower:   discard.NewGauge(),
		ByzantineValidators:      discard.NewGauge(),
		ByzantineValidatorsPower: discard.NewGauge(),

		BlockIntervalSeconds: discard.NewGauge(),

		NumTxs:                    discard.NewGauge(),
		BlockSizeBytes:            discard.NewGauge(),
		TotalTxs:                  discard.NewGauge(),
		CommittedHeight:           discard.NewGauge(),
		FastSyncing:               discard.NewGauge(),
		StateSyncing:              discard.NewGauge(),
		BlockParts:                discard.NewCounter(),
		QuorumPrevoteMessageDelay: discard.NewGauge(),
		FullPrevoteMessageDelay:   discard.NewGauge(),

		MissingProposal: discard.NewGauge(),
		RoundFailures:   discard.NewHistogram(),

		DurationProposal:           discard.NewHistogram(),
		DurationPrevote:            discard.NewHistogram(),
		DurationPrecommit:          discard.NewHistogram(),
		DurationCommitExecuting:    discard.NewHistogram(),
		DurationCommitCommitting:   discard.NewHistogram(),
		DurationCommitRechecking:   discard.NewHistogram(),
		DurationWaitingForNewRound: discard.NewHistogram(),

		DurationGaugeProposal:           discard.NewGauge(),
		DurationGaugePrevote:            discard.NewGauge(),
		DurationGaugePrecommit:          discard.NewGauge(),
		DurationGaugeCommitExecuting:    discard.NewGauge(),
		DurationGaugeCommitCommitting:   discard.NewGauge(),
		DurationGaugeCommitRechecking:   discard.NewGauge(),
		DurationGaugeWaitingForNewRound: discard.NewGauge(),
	}
}
