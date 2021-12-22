package evidence_test

import (
	"testing"
	"time"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/line/tm-db/v2/memdb"

	"github.com/stretchr/testify/mock"

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

func TestVerifyLightClientAttack_Lunatic(t *testing.T) {

	commonVals, commonVoters, commonPrivVals := types.RandVoterSet(2, 10)
	// use the correct Proof to bypass the checks in libsodium
	var proof []byte
	proof, err := commonPrivVals[0].GenerateVRFProof([]byte{})
	require.NoError(t, err)

	newVal, newPrivVal := types.RandValidatorForPrivKey(types.PrivKeyEd25519, false, 9)

	conflictingVals, err := types.ValidatorSetFromExistingValidators(append(commonVals.Validators, newVal))
	require.NoError(t, err)
	conflictingVoters, err := types.ValidatorSetFromExistingValidators(append(commonVoters.Voters, newVal))
	require.NoError(t, err)
	conflictingVoterSet := types.ToVoterAll(conflictingVoters.Validators)
	conflictingPrivVals := append(commonPrivVals, newPrivVal)

	commonHeader := makeHeaderRandom(4)
	commonHeader.Time = defaultEvidenceTime
	commonHeader.Proof = proof
	trustedHeader := makeHeaderRandom(10)
	trustedHeader.Time = defaultEvidenceTime.Add(1 * time.Hour)

	conflictingHeader := makeHeaderRandom(10)
	conflictingHeader.Time = defaultEvidenceTime.Add(1 * time.Hour)
	conflictingHeader.ValidatorsHash = conflictingVals.Hash()
	conflictingHeader.VotersHash = conflictingVoterSet.Hash()
	conflictingHeader.Proof = proof

	// we are simulating a lunatic light client attack
	blockID := makeBlockID(conflictingHeader.Hash(), 1000, []byte("partshash"))
	voteSet := types.NewVoteSet(evidenceChainID, 10, 1, tmproto.SignedMsgType(2), conflictingVoterSet)
	commit, err := types.MakeCommit(blockID, 10, 1, voteSet, conflictingPrivVals, defaultEvidenceTime)
	require.NoError(t, err)
	ev := &types.LightClientAttackEvidence{
		ConflictingBlock: &types.LightBlock{
			SignedHeader: &types.SignedHeader{
				Header: conflictingHeader,
				Commit: commit,
			},
			ValidatorSet: conflictingVals,
			VoterSet:     conflictingVoterSet,
		},
		CommonHeight:        4,
		TotalVotingPower:    20,
		ByzantineValidators: commonVals.Validators,
		Timestamp:           defaultEvidenceTime,
	}

	commonSignedHeader := &types.SignedHeader{
		Header: commonHeader,
		Commit: &types.Commit{},
	}
	commonSignedHeader.Proof = proof

	trustedBlockID := makeBlockID(trustedHeader.Hash(), 1000, []byte("partshash"))
	_, voters, privVals := types.RandVoterSet(3, 8)
	trustedVoteSet := types.NewVoteSet(evidenceChainID, 10, 1, tmproto.SignedMsgType(2), voters)
	trustedCommit, err := types.MakeCommit(trustedBlockID, 10, 1, trustedVoteSet, privVals, defaultEvidenceTime)
	require.NoError(t, err)
	trustedSignedHeader := &types.SignedHeader{
		Header: trustedHeader,
		Commit: trustedCommit,
	}

	// good pass -> no error
	err = evidence.VerifyLightClientAttack(
		ev,
		commonSignedHeader,
		trustedSignedHeader,
		commonVals,
		commonVoters,
		defaultEvidenceTime.Add(2*time.Hour),
		3*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.NoError(t, err)

	// trusted and conflicting hashes are the same -> an error should be returned
	err = evidence.VerifyLightClientAttack(
		ev,
		commonSignedHeader,
		ev.ConflictingBlock.SignedHeader,
		commonVals,
		commonVoters,
		defaultEvidenceTime.Add(2*time.Hour),
		3*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.Error(t, err)

	// evidence with different total validator power should fail
	ev.TotalVotingPower = 1
	err = evidence.VerifyLightClientAttack(
		ev,
		commonSignedHeader,
		trustedSignedHeader,
		commonVals,
		commonVoters,
		defaultEvidenceTime.Add(2*time.Hour),
		3*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.Error(t, err)
	ev.TotalVotingPower = 20

	forwardConflictingHeader := makeHeaderRandom(11)
	forwardConflictingHeader.Time = defaultEvidenceTime.Add(30 * time.Minute)
	forwardConflictingHeader.ValidatorsHash = conflictingVals.Hash()
	forwardConflictingHeader.VotersHash = conflictingVoterSet.Hash()
	forwardBlockID := makeBlockID(forwardConflictingHeader.Hash(), 1000, []byte("partshash"))
	forwardVoteSet := types.NewVoteSet(evidenceChainID, 11, 1, tmproto.SignedMsgType(2), conflictingVoterSet)
	forwardCommit, err := types.MakeCommit(forwardBlockID, 11, 1, forwardVoteSet, conflictingPrivVals, defaultEvidenceTime)
	require.NoError(t, err)
	forwardLunaticEv := &types.LightClientAttackEvidence{
		ConflictingBlock: &types.LightBlock{
			SignedHeader: &types.SignedHeader{
				Header: forwardConflictingHeader,
				Commit: forwardCommit,
			},
			ValidatorSet: conflictingVals,
			VoterSet:     conflictingVoterSet,
		},
		CommonHeight:        4,
		TotalVotingPower:    20,
		ByzantineValidators: commonVals.Validators,
		Timestamp:           defaultEvidenceTime,
	}
	err = evidence.VerifyLightClientAttack(
		forwardLunaticEv,
		commonSignedHeader,
		trustedSignedHeader,
		commonVals,
		commonVoters,
		defaultEvidenceTime.Add(2*time.Hour),
		3*time.Hour,
		types.DefaultVoterParams(),
	)
	assert.NoError(t, err)

	state := sm.State{
		LastBlockTime:   defaultEvidenceTime.Add(2 * time.Hour),
		LastBlockHeight: 11,
		ConsensusParams: *types.DefaultConsensusParams(),
		VoterParams:     types.DefaultVoterParams(),
	}
	stateStore := &smmocks.Store{}
	stateStore.On("LoadValidators", int64(4)).Return(commonVals, nil)
	stateStore.On("LoadVoters", int64(4), mock.AnythingOfType("*types.VoterParams")).Return(commonVoters, nil)
	stateStore.On("Load").Return(state, nil)
	blockStore := &mocks.BlockStore{}
	blockStore.On("LoadBlockMeta", int64(4)).Return(&types.BlockMeta{Header: *commonHeader})
	blockStore.On("LoadBlockMeta", int64(10)).Return(&types.BlockMeta{Header: *trustedHeader})
	blockStore.On("LoadBlockMeta", int64(11)).Return(nil)
	blockStore.On("LoadBlockCommit", int64(4)).Return(commit)
	blockStore.On("LoadBlockCommit", int64(10)).Return(trustedCommit)
	blockStore.On("Height").Return(int64(10))

	pool, err := evidence.NewPool(memdb.NewDB(), stateStore, blockStore)
	require.NoError(t, err)
	pool.SetLogger(log.TestingLogger())

	evList := types.EvidenceList{ev}
	err = pool.CheckEvidence(evList)
	assert.NoError(t, err)

	pendingEvs, _ := pool.PendingEvidence(state.ConsensusParams.Evidence.MaxBytes)
	assert.Equal(t, 1, len(pendingEvs))

	// if we submit evidence only against a single byzantine validator when we see there are more validators then this
	// should return an error
	ev.ByzantineValidators = []*types.Validator{commonVals.Validators[0]}
	err = pool.CheckEvidence(evList)
	assert.Error(t, err)
	ev.ByzantineValidators = commonVals.Validators // restore evidence

	// If evidence is submitted with an altered timestamp it should return an error
	ev.Timestamp = defaultEvidenceTime.Add(1 * time.Minute)
	err = pool.CheckEvidence(evList)
	assert.Error(t, err)

	evList = types.EvidenceList{forwardLunaticEv}
	err = pool.CheckEvidence(evList)
	assert.NoError(t, err)
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
		ByzantineValidators: conflictingVals.Validators[:4],
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
	}
	stateStore := &smmocks.Store{}
	stateStore.On("LoadValidators", int64(10)).Return(conflictingVals, nil)
	stateStore.On("LoadVoters", int64(10), mock.AnythingOfType("*types.VoterParams")).Return(conflictingVoters, nil)
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
	}
	stateStore := &smmocks.Store{}
	stateStore.On("LoadValidators", int64(10)).Return(conflictingVals, nil)
	stateStore.On("LoadVoters", int64(10), mock.AnythingOfType("*types.VoterParams")).Return(conflictingVoters, nil)
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
	}
	stateStore := &smmocks.Store{}
	stateStore.On("LoadVoters", int64(10), mock.AnythingOfType("*types.VoterParams")).Return(voterSet, nil)
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
