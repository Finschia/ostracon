package consensus

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"

	prometheus "github.com/go-kit/kit/metrics/prometheus"
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

	// VoterLastSignedHeight of a voter.
	VoterLastSignedHeight metrics.Gauge

	// Number of rounds.
	Rounds metrics.Gauge

	// ValidatorOrVoter: voter
	// Number of validators
	Validators metrics.Gauge
	// Total power of all validators.
	ValidatorsPower metrics.Gauge
	// Number of voters.
	Voters metrics.Gauge
	// Total power of all voters.
	VotersPower metrics.Gauge
	// Power of a voter.
	VoterPower metrics.Gauge
	// Amount of blocks missed by a voter.
	VoterMissedBlocks metrics.Gauge
	// Number of voters who did not sign.
	MissingVoters metrics.Gauge
	// Total power of the missing voters.
	MissingVotersPower metrics.Gauge
	// Number of voters who tried to double sign.
	ByzantineVoters metrics.Gauge
	// Total power of the byzantine voters.
	ByzantineVotersPower metrics.Gauge

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
		ValidatorsPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "validators_power",
			Help:      "Total power of all validators.",
		}, labels).With(labelsAndValues...),
		Voters: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "voters",
			Help:      "Number of voters.",
		}, labels).With(labelsAndValues...),
		VoterLastSignedHeight: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "voter_last_signed_height",
			Help:      "Last signed height for a voter",
		}, append(labels, "validator_address")).With(labelsAndValues...),
		VoterMissedBlocks: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "voter_missed_blocks",
			Help:      "Total missed blocks for a voter",
		}, append(labels, "validator_address")).With(labelsAndValues...),
		VotersPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "voters_power",
			Help:      "Total power of all voters.",
		}, labels).With(labelsAndValues...),
		VoterPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "voter_power",
			Help:      "Power of a voter",
		}, append(labels, "validator_address")).With(labelsAndValues...),
		MissingVoters: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "missing_voters",
			Help:      "Number of voters who did not sign.",
		}, labels).With(labelsAndValues...),
		MissingVotersPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "missing_voters_power",
			Help:      "Total power of the missing voters.",
		}, labels).With(labelsAndValues...),
		ByzantineVoters: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "byzantine_voters",
			Help:      "Number of voters who tried to double sign.",
		}, labels).With(labelsAndValues...),
		ByzantineVotersPower: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "byzantine_voters_power",
			Help:      "Total power of the byzantine voters.",
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

		VoterLastSignedHeight: discard.NewGauge(),

		Rounds: discard.NewGauge(),

		Validators:           discard.NewGauge(),
		ValidatorsPower:      discard.NewGauge(),
		Voters:               discard.NewGauge(),
		VotersPower:          discard.NewGauge(),
		VoterPower:           discard.NewGauge(),
		VoterMissedBlocks:    discard.NewGauge(),
		MissingVoters:        discard.NewGauge(),
		MissingVotersPower:   discard.NewGauge(),
		ByzantineVoters:      discard.NewGauge(),
		ByzantineVotersPower: discard.NewGauge(),

		BlockIntervalSeconds: discard.NewGauge(),

		NumTxs:          discard.NewGauge(),
		BlockSizeBytes:  discard.NewGauge(),
		TotalTxs:        discard.NewGauge(),
		CommittedHeight: discard.NewGauge(),
		FastSyncing:     discard.NewGauge(),
		StateSyncing:    discard.NewGauge(),
		BlockParts:      discard.NewCounter(),

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
