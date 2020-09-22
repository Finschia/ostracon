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
	BlockIntervalSeconds metrics.Histogram

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

	// Execution time profiling of each step
	ProposalWaiting         metrics.Histogram
	ProposalVerifying       metrics.Histogram
	ProposalBlockReceiving  metrics.Histogram
	PrevoteBlockVerifying   metrics.Histogram
	PrevoteReceiving        metrics.Histogram
	PrecommitBlockVerifying metrics.Histogram
	PrecommitReceiving      metrics.Histogram
	CommitBlockVerifying    metrics.Histogram
	CommitBlockApplying     metrics.Histogram
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
		BlockIntervalSeconds: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
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
		ProposalWaiting: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_proposal_waiting",
			Help:      "Duration between enterNewRound and receiving proposal",
			Buckets:   stdprometheus.LinearBuckets(1, 100, 10),
		}, labels).With(labelsAndValues...),
		ProposalVerifying: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_proposal_verifying",
			Help:      "Duration of ValidBlock",
			Buckets:   stdprometheus.LinearBuckets(1, 50, 10),
		}, labels).With(labelsAndValues...),
		ProposalBlockReceiving: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_proposal_block_receiving",
			Help:      "Duration of receiving all proposal block parts",
			Buckets:   stdprometheus.LinearBuckets(1, 100, 10),
		}, labels).With(labelsAndValues...),
		PrevoteBlockVerifying: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_prevote_block_verifying",
			Help:      "Duration of ValidBlock in prevote",
			Buckets:   stdprometheus.LinearBuckets(1, 50, 10),
		}, labels).With(labelsAndValues...),
		PrevoteReceiving: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_prevote_receiving",
			Help:      "Duration of receiving 2/3+ prevotes",
			Buckets:   stdprometheus.LinearBuckets(1, 100, 10),
		}, labels).With(labelsAndValues...),
		PrecommitBlockVerifying: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_precommit_block_verifying",
			Help:      "Duration of ValidBlock in precommit",
			Buckets:   stdprometheus.LinearBuckets(1, 50, 10),
		}, labels).With(labelsAndValues...),
		PrecommitReceiving: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_precommit_receiving",
			Help:      "Duration of receiving 2/3+ precommits",
			Buckets:   stdprometheus.LinearBuckets(1, 100, 10),
		}, labels).With(labelsAndValues...),
		CommitBlockVerifying: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_commit_block_verifying",
			Help:      "Duration of ValidBlock in commit",
			Buckets:   stdprometheus.LinearBuckets(1, 50, 10),
		}, labels).With(labelsAndValues...),
		CommitBlockApplying: prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "duration_commit_block_applying",
			Help:      "Duration of applying block",
			Buckets:   stdprometheus.LinearBuckets(1, 100, 10),
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

		BlockIntervalSeconds: discard.NewHistogram(),

		NumTxs:          discard.NewGauge(),
		BlockSizeBytes:  discard.NewGauge(),
		TotalTxs:        discard.NewGauge(),
		CommittedHeight: discard.NewGauge(),
		FastSyncing:     discard.NewGauge(),
		StateSyncing:    discard.NewGauge(),
		BlockParts:      discard.NewCounter(),

		ProposalWaiting:         discard.NewHistogram(),
		ProposalVerifying:       discard.NewHistogram(),
		ProposalBlockReceiving:  discard.NewHistogram(),
		PrevoteBlockVerifying:   discard.NewHistogram(),
		PrevoteReceiving:        discard.NewHistogram(),
		PrecommitBlockVerifying: discard.NewHistogram(),
		PrecommitReceiving:      discard.NewHistogram(),
		CommitBlockVerifying:    discard.NewHistogram(),
		CommitBlockApplying:     discard.NewHistogram(),
	}
}
