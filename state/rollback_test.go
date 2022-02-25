package state_test

import (
	"crypto/rand"
	"testing"

	dbm "github.com/line/tm-db/v2/memdb"
	"github.com/stretchr/testify/require"

	"github.com/line/ostracon/crypto/tmhash"
	tmstate "github.com/line/ostracon/proto/ostracon/state"
	tmversion "github.com/line/ostracon/proto/ostracon/version"
	"github.com/line/ostracon/state"
	"github.com/line/ostracon/state/mocks"
	"github.com/line/ostracon/types"
	"github.com/line/ostracon/version"
)

func TestRollback(t *testing.T) {
	stateStore := state.NewStore(dbm.NewDB())
	blockStore := &mocks.BlockStore{}
	var (
		height     int64  = 100
		appVersion uint64 = 10
	)

	proofHash := []byte{0}
	voterParams := types.DefaultVoterParams()
	previousValSet, _ := types.RandValidatorSet(5, 10)
	previousVoterSet := types.SelectVoter(previousValSet, proofHash, voterParams)
	lastValSet := previousValSet.CopyIncrementProposerPriority(1)
	lastVoterSet := types.SelectVoter(lastValSet, proofHash, voterParams)
	initialValSet := lastValSet.CopyIncrementProposerPriority(1)
	initialVoterSet := types.SelectVoter(initialValSet, proofHash, voterParams)
	nextValSet := initialValSet.CopyIncrementProposerPriority(1)
	nextVoterSet := types.SelectVoter(nextValSet, proofHash, voterParams)

	params := types.DefaultConsensusParams()
	params.Version.AppVersion = appVersion
	newParams := types.DefaultConsensusParams()
	newParams.Block.MaxBytes = 10000

	initialState := state.State{
		Version: tmstate.Version{
			Consensus: tmversion.Consensus{
				Block: version.BlockProtocol,
				App:   appVersion,
			},
			Software: version.OCCoreSemVer,
		},
		ChainID:                          "test-chain",
		InitialHeight:                    10,
		LastBlockID:                      makeBlockIDRandom(),
		AppHash:                          tmhash.Sum([]byte("app_hash")),
		LastResultsHash:                  tmhash.Sum([]byte("last_results_hash")),
		LastBlockHeight:                  height,
		LastVoters:                       lastVoterSet,
		LastProofHash:                    proofHash,
		Voters:                           initialVoterSet,
		VoterParams:                      voterParams,
		Validators:                       initialValSet,
		NextValidators:                   nextValSet,
		LastHeightValidatorsChanged:      height + 1,
		ConsensusParams:                  *params,
		LastHeightConsensusParamsChanged: height + 1,
	}
	previousState := initialState.Copy()
	previousState.LastBlockHeight = initialState.LastBlockHeight - 1
	previousState.LastHeightConsensusParamsChanged = initialState.LastHeightConsensusParamsChanged - 1
	previousState.LastHeightValidatorsChanged = initialState.LastHeightValidatorsChanged - 1
	previousState.LastVoters = previousVoterSet
	previousState.Voters = lastVoterSet
	previousState.Validators = lastValSet
	previousState.NextValidators = initialValSet
	require.NoError(t, stateStore.Bootstrap(previousState))
	require.NoError(t, stateStore.Bootstrap(initialState))

	height++
	block := &types.BlockMeta{
		Header: types.Header{
			Height:          height,
			AppHash:         initialState.AppHash,
			LastBlockID:     initialState.LastBlockID,
			LastResultsHash: initialState.LastResultsHash,
		},
	}
	blockStore.On("LoadBlockMeta", height).Return(block)

	appVersion++
	newParams.Version.AppVersion = appVersion
	nextState := initialState.Copy()
	nextState.LastBlockHeight = height
	nextState.Version.Consensus.App = appVersion
	nextState.LastBlockID = makeBlockIDRandom()
	nextState.AppHash = tmhash.Sum([]byte("next_app_hash"))
	nextState.LastVoters = initialVoterSet
	nextState.Voters = nextVoterSet
	nextState.Validators = nextValSet
	nextState.NextValidators = nextValSet.CopyIncrementProposerPriority(1)
	nextState.ConsensusParams = *newParams
	nextState.LastHeightConsensusParamsChanged = height + 1
	nextState.LastHeightValidatorsChanged = height + 1

	// update the state
	require.NoError(t, stateStore.Save(nextState))

	// rollback the state
	rollbackHeight, rollbackHash, err := state.Rollback(blockStore, stateStore)
	require.NoError(t, err)
	require.EqualValues(t, int64(100), rollbackHeight)
	require.EqualValues(t, initialState.AppHash, rollbackHash)
	blockStore.AssertExpectations(t)

	// assert that we've recovered the prior state
	loadedState, err := stateStore.Load()
	require.NoError(t, err)
	require.EqualValues(t, initialState, loadedState)
}

func TestRollbackNoState(t *testing.T) {
	stateStore := state.NewStore(dbm.NewDB())
	blockStore := &mocks.BlockStore{}

	_, _, err := state.Rollback(blockStore, stateStore)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no state found")
}

func TestRollbackNoBlocks(t *testing.T) {
	stateStore := state.NewStore(dbm.NewDB())
	blockStore := &mocks.BlockStore{}
	var (
		height     int64  = 100
		appVersion uint64 = 10
	)

	valSet, voterSet, _ := types.RandVoterSet(5, 10)

	params := types.DefaultConsensusParams()
	params.Version.AppVersion = appVersion
	newParams := types.DefaultConsensusParams()
	newParams.Block.MaxBytes = 10000

	initialState := state.State{
		Version: tmstate.Version{
			Consensus: tmversion.Consensus{
				Block: version.BlockProtocol,
				App:   10,
			},
			Software: version.OCCoreSemVer,
		},
		ChainID:                          "test-chain",
		InitialHeight:                    10,
		LastBlockID:                      makeBlockIDRandom(),
		AppHash:                          tmhash.Sum([]byte("app_hash")),
		LastResultsHash:                  tmhash.Sum([]byte("last_results_hash")),
		LastBlockHeight:                  height,
		LastVoters:                       voterSet,
		LastProofHash:                    []byte{0},
		Voters:                           voterSet,
		VoterParams:                      types.DefaultVoterParams(),
		Validators:                       valSet.CopyIncrementProposerPriority(1),
		NextValidators:                   valSet.CopyIncrementProposerPriority(2),
		LastHeightValidatorsChanged:      height + 1,
		ConsensusParams:                  *params,
		LastHeightConsensusParamsChanged: height + 1,
	}
	require.NoError(t, stateStore.Save(initialState))
	blockStore.On("LoadBlockMeta", height).Return(nil)

	_, _, err := state.Rollback(blockStore, stateStore)
	require.Error(t, err)
	require.Contains(t, err.Error(), "block at height 100 not found")
}

func makeBlockIDRandom() types.BlockID {
	var (
		blockHash   = make([]byte, tmhash.Size)
		partSetHash = make([]byte, tmhash.Size)
	)
	rand.Read(blockHash)   //nolint: errcheck // ignore errcheck for read
	rand.Read(partSetHash) //nolint: errcheck // ignore errcheck for read
	return types.BlockID{
		Hash: blockHash,
		PartSetHeader: types.PartSetHeader{
			Total: 123,
			Hash:  partSetHash,
		},
	}
}