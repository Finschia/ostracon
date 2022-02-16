package evidence_test

import (
	"bytes"
	"sort"
	"testing"
	"time"

	"github.com/line/ostracon/light"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/line/tm-db/v2/memdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/tmhash"
	"github.com/line/ostracon/evidence"
	"github.com/line/ostracon/evidence/mocks"
	"github.com/line/ostracon/libs/log"
	tmproto "github.com/line/ostracon/proto/ostracon/types"
	tmversion "github.com/line/ostracon/proto/ostracon/version"
	sm "github.com/line/ostracon/state"
	smmocks "github.com/line/ostracon/state/mocks"
	"github.com/line/ostracon/types"
	"github.com/line/ostracon/version"
)

const (
	defaultVotingPower = 10
)

func TestVerifyLightClientAttack_Lunatic(t *testing.T) {
	const (
		height       int64 = 10
		commonHeight int64 = 4
		totalVals          = 10
		byzVals            = 4
	)
	attackTime := defaultEvidenceTime.Add(1 * time.Hour)
	// create valid lunatic evidence
	ev, trusted, common := makeLunaticEvidence(
		t, height, commonHeight, totalVals, byzVals, totalVals-byzVals, defaultEvidenceTime, attackTime)
	require.NoError(t, ev.ValidateBasic())

	// good pass -> no error
	err := evidence.VerifyLightClientAttack(ev, common.SignedHeader, trusted.SignedHeader,
		common.ValidatorSet,
		ev.ConflictingBlock.VoterSet, // Should use correct VoterSet for bls.VerifyAggregatedSignature
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	assert.NoError(t, err)

	// trusted and conflicting hashes are the same -> an error should be returned
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, ev.ConflictingBlock.SignedHeader,
		common.ValidatorSet, common.VoterSet,
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	assert.Error(t, err)

	// evidence with different total validator power should fail
	ev.TotalVotingPower = 1 * defaultVotingPower
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, trusted.SignedHeader,
		common.ValidatorSet, common.VoterSet,
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	assert.Error(t, err)

	// evidence without enough malicious votes should fail
	ev, trusted, common = makeLunaticEvidence(
		t, height, commonHeight, totalVals, byzVals-1, totalVals-byzVals, defaultEvidenceTime, attackTime)
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, trusted.SignedHeader,
		common.ValidatorSet, common.VoterSet,
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	assert.Error(t, err)
}

func TestVerifyLightClientAttack_validateABCIEvidence(t *testing.T) {
	const (
		height             = int64(10)
		commonHeight int64 = 4
		totalVals          = 10
		byzVals            = 4
		votingPower        = defaultVotingPower
	)
	attackTime := defaultEvidenceTime.Add(1 * time.Hour)
	// create valid lunatic evidence
	ev, trusted, common := makeLunaticEvidence(
		t, height, commonHeight, totalVals, byzVals, totalVals-byzVals, defaultEvidenceTime, attackTime)
	require.NoError(t, ev.ValidateBasic())

	// good pass -> no error
	err := evidence.VerifyLightClientAttack(ev, common.SignedHeader, trusted.SignedHeader,
		common.ValidatorSet,
		ev.ConflictingBlock.VoterSet, // Should use correct VoterSet for bls.VerifyAggregatedSignature
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	assert.NoError(t, err)

	// illegal nil validators
	validator, _ := types.RandValidator(false, votingPower)
	ev.ByzantineValidators = []*types.Validator{validator}
	amnesiaHeader, err := types.SignedHeaderFromProto(ev.ConflictingBlock.SignedHeader.ToProto())
	require.NoError(t, err)
	amnesiaHeader.ProposerAddress = nil
	amnesiaHeader.Commit.Round = 2
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, amnesiaHeader, // illegal header
		common.ValidatorSet,
		ev.ConflictingBlock.VoterSet, // Should use correct VoterSet for bls.VerifyAggregatedSignature
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	require.Error(t, err)
	require.Equal(t, "expected nil validators from an amnesia light client attack but got 1", err.Error())

	// illegal byzantine validators
	equivocationHeader, err := types.SignedHeaderFromProto(ev.ConflictingBlock.SignedHeader.ToProto())
	require.NoError(t, err)
	equivocationHeader.ProposerAddress = nil
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, equivocationHeader, // illegal header
		common.ValidatorSet,
		ev.ConflictingBlock.VoterSet, // Should use correct VoterSet for bls.VerifyAggregatedSignature
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	require.Error(t, err)
	require.Equal(t, "expected 10 byzantine validators from evidence but got 1", err.Error())

	// illegal byzantine validator address
	_, phantomVoterSet, _ := types.RandVoterSet(totalVals, defaultVotingPower)
	ev.ByzantineValidators = phantomVoterSet.Voters
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, trusted.SignedHeader,
		common.ValidatorSet,
		ev.ConflictingBlock.VoterSet, // Should use correct VoterSet for bls.VerifyAggregatedSignature
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	require.Error(t, err)
	require.Contains(t, err.Error(), "evidence contained an unexpected byzantine validator address;")

	// illegal byzantine validator staking power
	phantomVoterSet = types.ToVoterAll(ev.ConflictingBlock.VoterSet.Voters)
	phantomVoterSet.Voters[0].StakingPower = votingPower + 1
	ev.ByzantineValidators = phantomVoterSet.Voters
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, trusted.SignedHeader,
		common.ValidatorSet,
		ev.ConflictingBlock.VoterSet, // Should use correct VoterSet for bls.VerifyAggregatedSignature
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	require.Error(t, err)
	require.Contains(t, err.Error(), "evidence contained unexpected byzantine validator staking power;")

	// illegal byzantine validator voting power
	phantomVoterSet = types.ToVoterAll(ev.ConflictingBlock.VoterSet.Voters)
	phantomVoterSet.Voters[0].VotingPower = votingPower + 1
	ev.ByzantineValidators = phantomVoterSet.Voters
	err = evidence.VerifyLightClientAttack(ev, common.SignedHeader, trusted.SignedHeader,
		common.ValidatorSet,
		ev.ConflictingBlock.VoterSet, // Should use correct VoterSet for bls.VerifyAggregatedSignature
		defaultEvidenceTime.Add(2*time.Hour), 3*time.Hour, types.DefaultVoterParams())
	require.Error(t, err)
	require.Contains(t, err.Error(), "evidence contained unexpected byzantine validator voting power;")
}

func TestVerify_LunaticAttackAgainstState(t *testing.T) {
	const (
		height       int64 = 10
		commonHeight int64 = 4
		totalVals          = 10
		byzVals            = 4
	)
	attackTime := defaultEvidenceTime.Add(1 * time.Hour)
	// create valid lunatic evidence
	ev, trusted, common := makeLunaticEvidence(
		t, height, commonHeight, totalVals, byzVals, totalVals-byzVals, defaultEvidenceTime, attackTime)

	// now we try to test verification against state
	state := sm.State{
		LastBlockTime:   defaultEvidenceTime.Add(2 * time.Hour),
		LastBlockHeight: height + 1,
		ConsensusParams: *types.DefaultConsensusParams(),
		VoterParams:     types.DefaultVoterParams(),
	}
	stateStore := &smmocks.Store{}
	stateStore.On("LoadValidators", commonHeight).Return(common.ValidatorSet, nil)
	// Should use correct VoterSet for bls.VerifyAggregatedSignature
	stateStore.On("LoadVoters", commonHeight, state.VoterParams).Return(
		common.ValidatorSet, ev.ConflictingBlock.VoterSet, state.VoterParams, state.LastProofHash, nil)
	stateStore.On("Load").Return(state, nil)
	blockStore := &mocks.BlockStore{}
	blockStore.On("LoadBlockMeta", commonHeight).Return(&types.BlockMeta{Header: *common.Header})
	blockStore.On("LoadBlockMeta", height).Return(&types.BlockMeta{Header: *trusted.Header})
	blockStore.On("LoadBlockCommit", commonHeight).Return(common.Commit)
	blockStore.On("LoadBlockCommit", height).Return(trusted.Commit)
	pool, err := evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)
	pool.SetLogger(log.TestingLogger())

	evList := types.EvidenceList{ev}
	// check that the evidence pool correctly verifies the evidence
	assert.NoError(t, pool.CheckEvidence(evList))

	// as it was not originally in the pending bucket, it should now have been added
	pendingEvs, _ := pool.PendingEvidence(state.ConsensusParams.Evidence.MaxBytes)
	require.Equal(t, 1, len(pendingEvs))
	assert.Equal(t, ev, pendingEvs[0])

	// if we submit evidence only against a single byzantine validator when we see there are more validators then this
	// should return an error
	ev.ByzantineValidators = ev.ByzantineValidators[:1]
	t.Log(evList)
	assert.Error(t, pool.CheckEvidence(evList))
	// restore original byz vals
	ev.ByzantineValidators = ev.GetByzantineValidators(common.VoterSet, trusted.SignedHeader)

	// duplicate evidence should be rejected
	evList = types.EvidenceList{ev, ev}
	pool, err = evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)
	assert.Error(t, pool.CheckEvidence(evList))

	// If evidence is submitted with an altered timestamp it should return an error
	ev.Timestamp = defaultEvidenceTime.Add(1 * time.Minute)
	pool, err = evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)
	assert.Error(t, pool.AddEvidence(ev))
	ev.Timestamp = defaultEvidenceTime

	// Evidence submitted with a different validator power should fail
	ev.TotalVotingPower = 1
	pool, err = evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)
	assert.Error(t, pool.AddEvidence(ev))
	ev.TotalVotingPower = common.VoterSet.TotalVotingPower()
}

func TestVerify_ForwardLunaticAttack(t *testing.T) {
	const (
		nodeHeight   int64 = 8
		attackHeight int64 = 10
		commonHeight int64 = 4
		totalVals          = 10
		byzVals            = 5
	)
	attackTime := defaultEvidenceTime.Add(1 * time.Hour)

	// create a forward lunatic attack
	ev, trusted, common := makeLunaticEvidence(
		t, attackHeight, commonHeight, totalVals, byzVals, totalVals-byzVals, defaultEvidenceTime, attackTime)

	// now we try to test verification against state
	state := sm.State{
		LastBlockTime:   defaultEvidenceTime.Add(2 * time.Hour),
		LastBlockHeight: nodeHeight,
		ConsensusParams: *types.DefaultConsensusParams(),
		VoterParams:     types.DefaultVoterParams(),
	}

	// modify trusted light block so that it is of a height less than the conflicting one
	trusted.Header.Height = state.LastBlockHeight
	trusted.Header.Time = state.LastBlockTime

	stateStore := &smmocks.Store{}
	stateStore.On("LoadValidators", commonHeight).Return(common.ValidatorSet, nil)
	// Should use correct VoterSet for bls.VerifyAggregatedSignature
	stateStore.On("LoadVoters", commonHeight, state.VoterParams).Return(
		common.ValidatorSet, ev.ConflictingBlock.VoterSet, state.VoterParams, state.LastProofHash, nil)
	stateStore.On("Load").Return(state, nil)
	blockStore := &mocks.BlockStore{}
	blockStore.On("LoadBlockMeta", commonHeight).Return(&types.BlockMeta{Header: *common.Header})
	blockStore.On("LoadBlockMeta", nodeHeight).Return(&types.BlockMeta{Header: *trusted.Header})
	blockStore.On("LoadBlockMeta", attackHeight).Return(nil)
	blockStore.On("LoadBlockCommit", commonHeight).Return(common.Commit)
	blockStore.On("LoadBlockCommit", nodeHeight).Return(trusted.Commit)
	blockStore.On("Height").Return(nodeHeight)
	pool, err := evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)

	// check that the evidence pool correctly verifies the evidence
	assert.NoError(t, pool.CheckEvidence(types.EvidenceList{ev}))

	// now we use a time which isn't able to contradict the FLA - thus we can't verify the evidence
	oldBlockStore := &mocks.BlockStore{}
	oldHeader := trusted.Header
	oldHeader.Time = defaultEvidenceTime
	oldBlockStore.On("LoadBlockMeta", commonHeight).Return(&types.BlockMeta{Header: *common.Header})
	oldBlockStore.On("LoadBlockMeta", nodeHeight).Return(&types.BlockMeta{Header: *oldHeader})
	oldBlockStore.On("LoadBlockMeta", attackHeight).Return(nil)
	oldBlockStore.On("LoadBlockCommit", commonHeight).Return(common.Commit)
	oldBlockStore.On("LoadBlockCommit", nodeHeight).Return(trusted.Commit)
	oldBlockStore.On("Height").Return(nodeHeight)
	require.Equal(t, defaultEvidenceTime, oldBlockStore.LoadBlockMeta(nodeHeight).Header.Time)

	pool, err = evidence.NewPool(memdb.NewDB(), stateStore, oldBlockStore)
	require.NoError(t, err)
	assert.Error(t, pool.CheckEvidence(types.EvidenceList{ev}))
}

func TestVerifyLightClientAttack_Equivocation(t *testing.T) {
	conflictingVals, conflictingVoters, conflictingPrivVals := types.RandVoterSet(5, 10)
	trustedHeader := makeHeaderRandom(10)

	conflictingHeader := makeHeaderRandom(10)
	conflictingHeader.VotersHash = conflictingVoters.Hash()

	trustedHeader.VotersHash = conflictingHeader.VotersHash
	trustedHeader.NextValidatorsHash = conflictingHeader.NextValidatorsHash
	trustedHeader.ConsensusHash = conflictingHeader.ConsensusHash
	trustedHeader.AppHash = conflictingHeader.AppHash
	trustedHeader.LastResultsHash = conflictingHeader.LastResultsHash

	// we are simulating a duplicate vote attack where all the validators in the conflictingVals set
	// except the last validator vote twice
	blockID := makeBlockID(conflictingHeader.Hash(), 1000, []byte("partshash"))
	voteSet := types.NewVoteSet(evidenceChainID, 10, 1, tmproto.SignedMsgType(2), conflictingVoters)
	commit, err := types.MakeCommit(blockID, 10, 1, voteSet, conflictingPrivVals[:4], defaultEvidenceTime)
	require.NoError(t, err)
	ev := &types.LightClientAttackEvidence{
		ConflictingBlock: &types.LightBlock{
			SignedHeader: &types.SignedHeader{
				Header: conflictingHeader,
				Commit: commit,
			},
			ValidatorSet: conflictingVals,
			VoterSet:     conflictingVoters,
		},
		CommonHeight:        10,
		ByzantineValidators: conflictingVoters.Voters[:4],
		TotalVotingPower:    50,
		Timestamp:           defaultEvidenceTime,
	}

	trustedBlockID := makeBlockID(trustedHeader.Hash(), 1000, []byte("partshash"))
	trustedVoteSet := types.NewVoteSet(evidenceChainID, 10, 1, tmproto.SignedMsgType(2), conflictingVoters)
	trustedCommit, err := types.MakeCommit(trustedBlockID, 10, 1, trustedVoteSet, conflictingPrivVals, defaultEvidenceTime)
	require.NoError(t, err)
	trustedSignedHeader := &types.SignedHeader{
		Header: trustedHeader,
		Commit: trustedCommit,
	}

	// good pass -> no error
	err = evidence.VerifyLightClientAttack(
		ev,
		trustedSignedHeader,
		trustedSignedHeader,
		conflictingVals,
		conflictingVoters,
		defaultEvidenceTime.Add(1*time.Minute),
		2*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.NoError(t, err)

	// trusted and conflicting hashes are the same -> an error should be returned
	err = evidence.VerifyLightClientAttack(
		ev,
		trustedSignedHeader,
		ev.ConflictingBlock.SignedHeader,
		conflictingVals,
		conflictingVoters,
		defaultEvidenceTime.Add(1*time.Minute),
		2*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.Error(t, err)

	// conflicting header has different next validators hash which should have been correctly derived from
	// the previous round
	ev.ConflictingBlock.Header.NextValidatorsHash = crypto.CRandBytes(tmhash.Size)
	err = evidence.VerifyLightClientAttack(ev, trustedSignedHeader, trustedSignedHeader, nil, nil,
		defaultEvidenceTime.Add(1*time.Minute), 2*time.Hour, types.DefaultVoterParams())
	assert.Error(t, err)
	// revert next validators hash
	ev.ConflictingBlock.Header.NextValidatorsHash = trustedHeader.NextValidatorsHash

	state := sm.State{
		LastBlockTime:   defaultEvidenceTime.Add(1 * time.Minute),
		LastBlockHeight: 11,
		ConsensusParams: *types.DefaultConsensusParams(),
		VoterParams:     types.DefaultVoterParams(),
	}
	stateStore := &smmocks.Store{}
	stateStore.On("LoadValidators", int64(10)).Return(conflictingVals, nil)
	// Should use correct VoterSet for bls.VerifyAggregatedSignature
	stateStore.On("LoadVoters", int64(10), state.VoterParams).Return(
		conflictingVals, conflictingVoters, state.VoterParams, state.LastProofHash, nil)
	stateStore.On("Load").Return(state, nil)
	blockStore := &mocks.BlockStore{}
	blockStore.On("LoadBlockMeta", int64(10)).Return(&types.BlockMeta{Header: *trustedHeader})
	blockStore.On("LoadBlockCommit", int64(10)).Return(trustedCommit)

	pool, err := evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)
	pool.SetLogger(log.TestingLogger())

	evList := types.EvidenceList{ev}
	err = pool.CheckEvidence(evList)
	assert.NoError(t, err)

	pendingEvs, _ := pool.PendingEvidence(state.ConsensusParams.Evidence.MaxBytes)
	assert.Equal(t, 1, len(pendingEvs))
}

func TestVerifyLightClientAttack_Amnesia(t *testing.T) {
	conflictingVals, conflictingVoters, conflictingPrivVals := types.RandVoterSet(5, 10)

	conflictingHeader := makeHeaderRandom(10)
	conflictingHeader.VotersHash = conflictingVoters.Hash()
	trustedHeader := makeHeaderRandom(10)
	trustedHeader.VotersHash = conflictingHeader.VotersHash
	trustedHeader.NextValidatorsHash = conflictingHeader.NextValidatorsHash
	trustedHeader.AppHash = conflictingHeader.AppHash
	trustedHeader.ConsensusHash = conflictingHeader.ConsensusHash
	trustedHeader.LastResultsHash = conflictingHeader.LastResultsHash

	// we are simulating an amnesia attack where all the validators in the conflictingVals set
	// except the last validator vote twice. However this time the commits are of different rounds.
	blockID := makeBlockID(conflictingHeader.Hash(), 1000, []byte("partshash"))
	voteSet := types.NewVoteSet(evidenceChainID, 10, 0, tmproto.SignedMsgType(2), conflictingVoters)
	commit, err := types.MakeCommit(blockID, 10, 0, voteSet, conflictingPrivVals, defaultEvidenceTime)
	require.NoError(t, err)
	ev := &types.LightClientAttackEvidence{
		ConflictingBlock: &types.LightBlock{
			SignedHeader: &types.SignedHeader{
				Header: conflictingHeader,
				Commit: commit,
			},
			ValidatorSet: conflictingVals,
			VoterSet:     conflictingVoters,
		},
		CommonHeight:        10,
		ByzantineValidators: nil, // with amnesia evidence no validators are submitted as abci evidence
		TotalVotingPower:    50,
		Timestamp:           defaultEvidenceTime,
	}

	trustedBlockID := makeBlockID(trustedHeader.Hash(), 1000, []byte("partshash"))
	trustedVoteSet := types.NewVoteSet(evidenceChainID, 10, 1, tmproto.SignedMsgType(2), conflictingVoters)
	trustedCommit, err := types.MakeCommit(trustedBlockID, 10, 1, trustedVoteSet, conflictingPrivVals, defaultEvidenceTime)
	require.NoError(t, err)
	trustedSignedHeader := &types.SignedHeader{
		Header: trustedHeader,
		Commit: trustedCommit,
	}

	// good pass -> no error
	err = evidence.VerifyLightClientAttack(
		ev,
		trustedSignedHeader,
		trustedSignedHeader,
		conflictingVals,
		conflictingVoters,
		defaultEvidenceTime.Add(1*time.Minute),
		2*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.NoError(t, err)

	// trusted and conflicting hashes are the same -> an error should be returned
	err = evidence.VerifyLightClientAttack(
		ev,
		trustedSignedHeader,
		ev.ConflictingBlock.SignedHeader,
		conflictingVals,
		conflictingVoters,
		defaultEvidenceTime.Add(1*time.Minute),
		2*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.Error(t, err)

	state := sm.State{
		LastBlockTime:   defaultEvidenceTime.Add(1 * time.Minute),
		LastBlockHeight: 11,
		ConsensusParams: *types.DefaultConsensusParams(),
		VoterParams:     types.DefaultVoterParams(),
	}
	stateStore := &smmocks.Store{}
	stateStore.On("LoadValidators", int64(10)).Return(conflictingVals, nil)
	// Should use correct VoterSet for bls.VerifyAggregatedSignature
	stateStore.On("LoadVoters", int64(10), state.VoterParams).Return(
		conflictingVals, conflictingVoters, state.VoterParams, state.LastProofHash, nil)
	stateStore.On("Load").Return(state, nil)
	blockStore := &mocks.BlockStore{}
	blockStore.On("LoadBlockMeta", int64(10)).Return(&types.BlockMeta{Header: *trustedHeader})
	blockStore.On("LoadBlockCommit", int64(10)).Return(trustedCommit)

	pool, err := evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)
	pool.SetLogger(log.TestingLogger())

	evList := types.EvidenceList{ev}
	err = pool.CheckEvidence(evList)
	assert.NoError(t, err)

	pendingEvs, _ := pool.PendingEvidence(state.ConsensusParams.Evidence.MaxBytes)
	assert.Equal(t, 1, len(pendingEvs))
}

type voteData struct {
	vote1 *types.Vote
	vote2 *types.Vote
	valid bool
}

func TestVerifyDuplicateVoteEvidence(t *testing.T) {
	val := types.NewMockPV(types.PrivKeyComposite) // TODO 🏺 need to test by all key types
	val2 := types.NewMockPV(types.PrivKeyComposite)
	valSet := types.NewValidatorSet([]*types.Validator{val.ExtractIntoValidator(1)})
	voterSet := types.ToVoterAll(valSet.Validators)

	blockID := makeBlockID([]byte("blockhash"), 1000, []byte("partshash"))
	blockID2 := makeBlockID([]byte("blockhash2"), 1000, []byte("partshash"))
	blockID3 := makeBlockID([]byte("blockhash"), 10000, []byte("partshash"))
	blockID4 := makeBlockID([]byte("blockhash"), 10000, []byte("partshash2"))

	const chainID = "mychain"

	vote1 := makeVote(t, val, chainID, 0, 10, 2, 1, blockID, defaultEvidenceTime)
	v1 := vote1.ToProto()
	err := val.SignVote(chainID, v1)
	require.NoError(t, err)
	badVote := makeVote(t, val, chainID, 0, 10, 2, 1, blockID, defaultEvidenceTime)
	bv := badVote.ToProto()
	err = val2.SignVote(chainID, bv)
	require.NoError(t, err)

	vote1.Signature = v1.Signature
	badVote.Signature = bv.Signature

	cases := []voteData{
		{vote1, makeVote(t, val, chainID, 0, 10, 2, 1, blockID2, defaultEvidenceTime), true}, // different block ids
		{vote1, makeVote(t, val, chainID, 0, 10, 2, 1, blockID3, defaultEvidenceTime), true},
		{vote1, makeVote(t, val, chainID, 0, 10, 2, 1, blockID4, defaultEvidenceTime), true},
		{vote1, makeVote(t, val, chainID, 0, 10, 2, 1, blockID, defaultEvidenceTime), false},     // wrong block id
		{vote1, makeVote(t, val, "mychain2", 0, 10, 2, 1, blockID2, defaultEvidenceTime), false}, // wrong chain id
		{vote1, makeVote(t, val, chainID, 0, 11, 2, 1, blockID2, defaultEvidenceTime), false},    // wrong height
		{vote1, makeVote(t, val, chainID, 0, 10, 3, 1, blockID2, defaultEvidenceTime), false},    // wrong round
		{vote1, makeVote(t, val, chainID, 0, 10, 2, 2, blockID2, defaultEvidenceTime), false},    // wrong step
		{vote1, makeVote(t, val2, chainID, 0, 10, 2, 1, blockID2, defaultEvidenceTime), false},   // wrong validator
		// a different vote time doesn't matter
		{vote1, makeVote(t, val, chainID, 0, 10, 2, 1, blockID2, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)), true},
		{vote1, badVote, false}, // signed by wrong key
	}

	require.NoError(t, err)
	for _, c := range cases {
		ev := &types.DuplicateVoteEvidence{
			VoteA:            c.vote1,
			VoteB:            c.vote2,
			ValidatorPower:   1,
			TotalVotingPower: 1,
			Timestamp:        defaultEvidenceTime,
		}
		if c.valid {
			assert.Nil(t, evidence.VerifyDuplicateVote(ev, chainID, voterSet), "evidence should be valid")
		} else {
			assert.NotNil(t, evidence.VerifyDuplicateVote(ev, chainID, voterSet), "evidence should be invalid")
		}
	}

	// create good evidence and correct validator power
	goodEv := types.NewMockDuplicateVoteEvidenceWithValidator(10, defaultEvidenceTime, val, chainID)
	goodEv.ValidatorPower = 1
	goodEv.TotalVotingPower = 1
	badEv := types.NewMockDuplicateVoteEvidenceWithValidator(10, defaultEvidenceTime, val, chainID)
	badTimeEv := types.NewMockDuplicateVoteEvidenceWithValidator(10, defaultEvidenceTime.Add(1*time.Minute), val, chainID)
	badTimeEv.ValidatorPower = 1
	badTimeEv.TotalVotingPower = 1
	state := sm.State{
		ChainID:         chainID,
		LastBlockTime:   defaultEvidenceTime.Add(1 * time.Minute),
		LastBlockHeight: 11,
		ConsensusParams: *types.DefaultConsensusParams(),
		VoterParams:     types.DefaultVoterParams(),
	}
	stateStore := &smmocks.Store{}
	// Should use correct VoterSet for bls.VerifyAggregatedSignature
	stateStore.On("LoadVoters", int64(10), state.VoterParams).Return(
		valSet, voterSet, state.VoterParams, state.LastProofHash, nil)
	stateStore.On("Load").Return(state, nil)
	blockStore := &mocks.BlockStore{}
	blockStore.On("LoadBlockMeta", int64(10)).Return(&types.BlockMeta{Header: types.Header{Time: defaultEvidenceTime}})

	pool, err := evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)

	evList := types.EvidenceList{goodEv}
	err = pool.CheckEvidence(evList)
	assert.NoError(t, err)

	// evidence with a different validator power should fail
	evList = types.EvidenceList{badEv}
	err = pool.CheckEvidence(evList)
	assert.Error(t, err)

	// evidence with a different timestamp should fail
	evList = types.EvidenceList{badTimeEv}
	err = pool.CheckEvidence(evList)
	assert.Error(t, err)
}

func makeLunaticEvidence(
	t *testing.T,
	height, commonHeight int64,
	totalVals, byzVals, phantomVals int,
	commonTime, attackTime time.Time,
) (ev *types.LightClientAttackEvidence, trusted *types.LightBlock, common *types.LightBlock) {
	commonValSet, commonVoterSet, commonPrivVals := types.RandVoterSet(totalVals, defaultVotingPower)
	require.Greater(t, totalVals, byzVals)

	// use the correct Proof to bypass the checks in libsodium
	var proof []byte
	proof, err := commonPrivVals[0].GenerateVRFProof([]byte{})
	require.NoError(t, err)

	// extract out the subset of byzantine validators in the common validator set
	byzValidators, byzVoters, byzPrivVals :=
		commonValSet.Validators[:byzVals], commonVoterSet.Voters[:byzVals], commonPrivVals[:byzVals]

	phantomValSet, phantomVoterSet, phantomPrivVals := types.RandVoterSet(phantomVals, defaultVotingPower)

	conflictingVals := types.NewValidatorSet(append(phantomValSet.Validators, byzValidators...))
	conflictingVoters := types.ToVoterAll(append(phantomVoterSet.Voters, byzVoters...))
	conflictingPrivVals := append(phantomPrivVals, byzPrivVals...)

	// Should be the same order between VoterSet.Voters and PrivValidators for Commit
	conflictingPrivVals = orderPrivValsByVoterSet(t, conflictingVoters, conflictingPrivVals)

	conflictingHeader := makeHeaderRandom(height)
	conflictingHeader.Proof = proof
	conflictingHeader.Time = attackTime
	conflictingHeader.ValidatorsHash = conflictingVals.Hash()
	conflictingHeader.VotersHash = conflictingVoters.Hash()

	blockID := makeBlockID(conflictingHeader.Hash(), 1000, []byte("partshash"))
	voteSet := types.NewVoteSet(evidenceChainID, height, 1, tmproto.SignedMsgType(2), conflictingVoters)
	commit, err := types.MakeCommit(blockID, height, 1, voteSet, conflictingPrivVals, defaultEvidenceTime)
	require.NoError(t, err)
	err = conflictingVoters.VerifyCommitLightTrusting(evidenceChainID, commit, light.DefaultTrustLevel)
	require.NoError(t, err)
	byzantineValidators := conflictingVoters.Copy().Voters
	sort.Sort(types.ValidatorsByVotingPower(byzantineValidators))
	ev = &types.LightClientAttackEvidence{
		ConflictingBlock: &types.LightBlock{
			SignedHeader: &types.SignedHeader{
				Header: conflictingHeader,
				Commit: commit,
			},
			ValidatorSet: conflictingVals,
			VoterSet:     conflictingVoters,
		},
		CommonHeight:        commonHeight,
		TotalVotingPower:    commonVoterSet.TotalVotingPower(),
		ByzantineValidators: byzantineValidators,
		Timestamp:           commonTime,
	}

	trustedHeader := makeHeaderRandom(height)
	trustedHeader.Proof = proof
	trustedBlockID := makeBlockID(trustedHeader.Hash(), 1000, []byte("partshash"))
	trustedVals, trustedVoters, trustedPrivVals := types.RandVoterSet(totalVals, defaultVotingPower)

	// Should be the same order between VoterSet.Voters and PrivValidators for Commit
	trustedPrivVals = orderPrivValsByVoterSet(t, trustedVoters, trustedPrivVals)

	trustedVoteSet := types.NewVoteSet(evidenceChainID, height, 1, tmproto.SignedMsgType(2), trustedVoters)
	trustedCommit, err := types.MakeCommit(trustedBlockID, height, 1, trustedVoteSet, trustedPrivVals, defaultEvidenceTime)
	require.NoError(t, err)
	err = trustedVoters.VerifyCommitLightTrusting(evidenceChainID, trustedCommit, light.DefaultTrustLevel)
	require.NoError(t, err)
	trusted = &types.LightBlock{
		SignedHeader: &types.SignedHeader{
			Header: trustedHeader,
			Commit: trustedCommit,
		},
		ValidatorSet: trustedVals,
		VoterSet:     trustedVoters,
	}

	commonHeader := makeHeaderRandom(commonHeight)
	commonHeader.Proof = proof
	commonHeader.Time = commonTime
	common = &types.LightBlock{
		SignedHeader: &types.SignedHeader{
			Header: commonHeader,
			// we can leave this empty because we shouldn't be checking this
			Commit: &types.Commit{},
		},
		ValidatorSet: commonValSet,
		VoterSet:     commonVoterSet,
	}

	return ev, trusted, common
}

// func makeEquivocationEvidence() *types.LightClientAttackEvidence {

// }

// func makeAmnesiaEvidence() *types.LightClientAttackEvidence {

// }

func makeVote(
	t *testing.T, val types.PrivValidator, chainID string, valIndex int32, height int64,
	round int32, step int, blockID types.BlockID, time time.Time) *types.Vote {
	pubKey, err := val.GetPubKey()
	require.NoError(t, err)
	v := &types.Vote{
		ValidatorAddress: pubKey.Address(),
		ValidatorIndex:   valIndex,
		Height:           height,
		Round:            round,
		Type:             tmproto.SignedMsgType(step),
		BlockID:          blockID,
		Timestamp:        time,
		Signature:        []byte{},
	}

	vpb := v.ToProto()
	err = val.SignVote(chainID, vpb)
	if err != nil {
		panic(err)
	}
	v.Signature = vpb.Signature
	return v
}

func makeHeaderRandom(height int64) *types.Header {
	return &types.Header{
		Version:            tmversion.Consensus{Block: version.BlockProtocol, App: 1},
		ChainID:            evidenceChainID,
		Height:             height,
		Time:               defaultEvidenceTime,
		LastBlockID:        makeBlockID([]byte("headerhash"), 1000, []byte("partshash")),
		LastCommitHash:     crypto.CRandBytes(tmhash.Size),
		DataHash:           crypto.CRandBytes(tmhash.Size),
		VotersHash:         crypto.CRandBytes(tmhash.Size),
		NextValidatorsHash: crypto.CRandBytes(tmhash.Size),
		ConsensusHash:      crypto.CRandBytes(tmhash.Size),
		AppHash:            crypto.CRandBytes(tmhash.Size),
		LastResultsHash:    crypto.CRandBytes(tmhash.Size),
		EvidenceHash:       crypto.CRandBytes(tmhash.Size),
		ProposerAddress:    crypto.CRandBytes(crypto.AddressSize),
		Proof:              crypto.CRandBytes(vrf.ProofSize),
	}
}

func makeBlockID(hash []byte, partSetSize uint32, partSetHash []byte) types.BlockID {
	var (
		h   = make([]byte, tmhash.Size)
		psH = make([]byte, tmhash.Size)
	)
	copy(h, hash)
	copy(psH, partSetHash)
	return types.BlockID{
		Hash: h,
		PartSetHeader: types.PartSetHeader{
			Total: partSetSize,
			Hash:  psH,
		},
	}
}

func orderPrivValsByVoterSet(
	t *testing.T, voters *types.VoterSet, privVals []types.PrivValidator) []types.PrivValidator {
	output := make([]types.PrivValidator, len(privVals))
	for idx, v := range voters.Voters {
		for _, p := range privVals {
			pubKey, err := p.GetPubKey()
			require.NoError(t, err)
			if bytes.Equal(v.Address, pubKey.Address()) {
				output[idx] = p
				break
			}
		}
		require.NotEmpty(t, output[idx])
	}
	return output
}
