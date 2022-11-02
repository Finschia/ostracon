package core

import (
	"fmt"
	"os"
	"testing"

	cfg "github.com/line/ostracon/config"
	"github.com/line/ostracon/consensus"
	ctypes "github.com/line/ostracon/rpc/core/types"
	rpctypes "github.com/line/ostracon/rpc/jsonrpc/types"
	sm "github.com/line/ostracon/state"
	"github.com/line/ostracon/state/mocks"
	"github.com/line/ostracon/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"
)

type args struct {
	ctx        *rpctypes.Context
	heightPtr  *int64
	pagePtr    *int
	perPagePtr *int
}

var (
	height            = int64(1)
	page              = 1
	perPage           = 10
	normalArgs        = args{&rpctypes.Context{}, &height, &page, &perPage}
	invalidHeight     = height + 10000
	invalidPage       = page + 10
	invalidHeightArgs = args{&rpctypes.Context{}, &invalidHeight, &page, &perPage}
	invalidPageArgs   = args{&rpctypes.Context{}, &height, &invalidPage, &perPage}
	noErrorFunc       = func(t assert.TestingT, err error, i ...interface{}) bool {
		return err == nil
	}
	errorFunc = func(t assert.TestingT, err error, i ...interface{}) bool {
		return err != nil
	}
)

func makeTestStateStore(t *testing.T) (sm.State, func()) {
	stateStore := sm.NewStore(dbm.NewMemDB())
	blockStore := &mocks.BlockStore{}

	config := cfg.ResetTestRoot("rpc_core_test")
	state, err := stateStore.LoadFromDBOrGenesisFile(config.GenesisFile())
	require.NoError(t, err)
	consensusState := consensus.NewState(
		config.Consensus, state, nil, blockStore, nil, nil)
	consensusReactor := consensus.NewReactor(consensusState, false, false, 0)

	val, _ := types.RandValidator(true, 10)
	vals := types.NewValidatorSet([]*types.Validator{val})

	state.Validators = vals
	state.Voters = types.SelectVoter(vals, state.LastProofHash, types.DefaultVoterParams())
	err = stateStore.Save(state)
	require.NoError(t, err)

	blockStore.On("Base").Return(state.LastBlockHeight)
	state.LastBlockHeight = state.LastBlockHeight + 1
	blockStore.On("Height").Return(state.LastBlockHeight)

	env = &Environment{}
	env.StateStore = stateStore
	env.BlockStore = blockStore
	env.ConsensusReactor = consensusReactor

	return state, func() { os.RemoveAll(config.RootDir) }
}

func TestVoters(t *testing.T) {
	state, cleanup := makeTestStateStore(t)
	defer cleanup()

	normalResult := &ctypes.ResultVoters{
		BlockHeight: height,
		Voters:      state.Voters.Voters,
		Count:       len(state.Voters.Voters),
		Total:       len(state.Voters.Voters),
	}

	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultVoters
		wantErr assert.ErrorAssertionFunc
	}{
		{"success", normalArgs, normalResult, noErrorFunc},
		{"invalid height", invalidHeightArgs, nil, errorFunc},
		{"invalid page", invalidPageArgs, nil, errorFunc},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Voters(tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)
			if !tt.wantErr(t, err, fmt.Sprintf("Voters(%v, %v, %v, %v)",
				tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Voters(%v, %v, %v, %v)",
				tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)
		})
	}
}

func TestValidatorsWithVoters(t *testing.T) {
	state, cleanup := makeTestStateStore(t)
	defer cleanup()

	vals := state.Validators.Validators
	validators := make([]*types.Validator, 0, len(vals))
	indices := make([]int32, 0, len(vals))
	for _, validator := range state.Validators.Validators {
		index, voter := state.Voters.GetByAddress(validator.Address)
		if index == -1 {
			validators = append(validators, validator)
		} else {
			indices = append(indices, index)
			validators = append(validators, voter)
		}
	}

	normalResult := &ctypes.ResultValidatorsWithVoters{
		BlockHeight:  height,
		Validators:   validators,
		Count:        len(validators),
		Total:        len(validators),
		VoterIndices: indices,
	}

	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultValidatorsWithVoters
		wantErr assert.ErrorAssertionFunc
	}{
		{"success", normalArgs, normalResult, noErrorFunc},
		{"invalid height", invalidHeightArgs, nil, errorFunc},
		{"invalid page", invalidPageArgs, nil, errorFunc},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatorsWithVoters(tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)
			if !tt.wantErr(t, err, fmt.Sprintf("ValidatorsWithVoters(%v, %v, %v, %v)",
				tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ValidatorsWithVoters(%v, %v, %v, %v)",
				tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)
		})
	}
}
