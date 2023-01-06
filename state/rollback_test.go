package state_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	dbm "github.com/tendermint/tm-db"

	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/tmhash"
	tmstate "github.com/line/ostracon/proto/ostracon/state"
	"github.com/line/ostracon/state"
	"github.com/line/ostracon/state/mocks"
	"github.com/line/ostracon/types"
	"github.com/line/ostracon/version"
)

var (
	proofHash = []byte{0}
)

func TestRollback(t *testing.T) {
	var (
		height     int64 = 100
		nextHeight int64 = 101
	)
	blockStore := &mocks.BlockStore{}
	stateStore := setupStateStore(t, height)
	initialState, err := stateStore.Load()
	require.NoError(t, err)

	// perform the rollback over a version bump
	newParams := types.DefaultConsensusParams()
	newParams.Version.AppVersion = 11
	newParams.Block.MaxBytes = 1000
	nextState := initialState.Copy()
	nextState.LastBlockHeight = nextHeight
	nextState.Version.Consensus.App = 11
	nextState.LastBlockID = makeBlockIDRandom()
	nextState.AppHash = tmhash.Sum([]byte("app_hash"))
	nextState.LastValidators = initialState.Validators
	nextState.Validators = initialState.NextValidators
	nextState.NextValidators = initialState.NextValidators.CopyIncrementProposerPriority(1)
	nextState.ConsensusParams = *newParams
	nextState.LastHeightConsensusParamsChanged = nextHeight + 1
	nextState.LastHeightValidatorsChanged = nextHeight + 1

	// update the state
	require.NoError(t, stateStore.Save(nextState))

	block := &types.BlockMeta{
		BlockID: initialState.LastBlockID,
		Header: types.Header{
			Height:          initialState.LastBlockHeight,
			AppHash:         crypto.CRandBytes(tmhash.Size),
			LastBlockID:     makeBlockIDRandom(),
			LastResultsHash: initialState.LastResultsHash,
		},
	}
	nextBlock := &types.BlockMeta{
		BlockID: initialState.LastBlockID,
		Header: types.Header{
			Height:          nextState.LastBlockHeight,
			AppHash:         initialState.AppHash,
			LastBlockID:     block.BlockID,
			LastResultsHash: nextState.LastResultsHash,
		},
	}
	blockStore.On("LoadBlockMeta", height).Return(block)
	blockStore.On("LoadBlockMeta", nextHeight).Return(nextBlock)
	blockStore.On("Height").Return(nextHeight)

	// rollback the state
	rollbackHeight, rollbackHash, err := state.Rollback(blockStore, stateStore)
	require.NoError(t, err)
	require.EqualValues(t, height, rollbackHeight)
	require.EqualValues(t, initialState.AppHash, rollbackHash)
	blockStore.AssertExpectations(t)

	// assert that we've recovered the prior state
	loadedState, err := stateStore.Load()
	require.NoError(t, err)
	require.EqualValues(t, initialState, loadedState)
}

func TestRollbackNoState(t *testing.T) {
	stateStore := state.NewStore(dbm.NewMemDB())
	blockStore := &mocks.BlockStore{}

	_, _, err := state.Rollback(blockStore, stateStore)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no state found")
}

func TestRollbackNoBlocks(t *testing.T) {
	const height = int64(100)
	stateStore := setupStateStore(t, height)
	blockStore := &mocks.BlockStore{}
	blockStore.On("Height").Return(height)
	blockStore.On("LoadBlockMeta", height-1).Once().Return(nil)

	_, _, err := state.Rollback(blockStore, stateStore)
	require.Error(t, err)
	require.Contains(t, err.Error(), "block at RollbackHeight 99 not found")

	blockStore.On("LoadBlockMeta", height-1).Once().Return(&types.BlockMeta{})
	blockStore.On("LoadBlockMeta", height).Return(nil)

	_, _, err = state.Rollback(blockStore, stateStore)
	require.Error(t, err)
	require.Contains(t, err.Error(), "block at LastBlockHeight 100 not found")
}

func TestRollbackDifferentStateHeight(t *testing.T) {
	const height = int64(100)
	stateStore := setupStateStore(t, height)
	blockStore := &mocks.BlockStore{}
	blockStore.On("Height").Return(height + 2)

	_, _, err := state.Rollback(blockStore, stateStore)
	require.Error(t, err)
	require.Equal(t, err.Error(), "statestore height (100) is not one below or equal to blockstore height (102)")
}

func setupStateStore(t *testing.T, height int64) state.Store {
	stateStore := state.NewStore(dbm.NewMemDB())

	previousValSet, _ := types.RandValidatorSet(5, 10)
	lastValSet := previousValSet.CopyIncrementProposerPriority(1)
	initialValSet := lastValSet.CopyIncrementProposerPriority(1)
	nextValSet := initialValSet.CopyIncrementProposerPriority(1)

	params := types.DefaultConsensusParams()
	params.Version.AppVersion = 10

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
		LastValidators:                   lastValSet,
		Validators:                       initialValSet,
		NextValidators:                   nextValSet,
		LastHeightValidatorsChanged:      height + 1,
		ConsensusParams:                  *params,
		LastHeightConsensusParamsChanged: height + 1,
		LastProofHash:                    proofHash,
	}

	// Need to set previous initial state for VRF verify
	previousState := initialState.Copy()
	previousState.LastBlockHeight = initialState.LastBlockHeight - 1
	previousState.LastHeightConsensusParamsChanged = initialState.LastHeightConsensusParamsChanged - 1
	previousState.LastHeightValidatorsChanged = initialState.LastHeightValidatorsChanged - 1
	previousState.LastValidators = previousValSet
	previousState.Validators = lastValSet
	previousState.NextValidators = initialValSet
	require.NoError(t, stateStore.Bootstrap(previousState))

	require.NoError(t, stateStore.Bootstrap(initialState))
	return stateStore
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
