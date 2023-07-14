package core

import (
	"fmt"
	"os"
	"testing"

	cfg "github.com/Finschia/ostracon/config"
	"github.com/Finschia/ostracon/consensus"
	ctypes "github.com/Finschia/ostracon/rpc/core/types"
	rpctypes "github.com/Finschia/ostracon/rpc/jsonrpc/types"
	sm "github.com/Finschia/ostracon/state"
	"github.com/Finschia/ostracon/state/mocks"
	"github.com/Finschia/ostracon/types"
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
	stateStore := sm.NewStore(dbm.NewMemDB(), sm.StoreOptions{
		DiscardABCIResponses: false,
	})
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

func TestValidators(t *testing.T) {
	state, cleanup := makeTestStateStore(t)
	defer cleanup()

	normalResult := &ctypes.ResultValidators{
		Validators:  state.Validators.Validators,
		BlockHeight: height,
		Total:       len(state.Validators.Validators),
		Count:       len(state.Validators.Validators),
	}

	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
		want    *ctypes.ResultValidators
	}{
		{"success", normalArgs, noErrorFunc, normalResult},
		{"invalid page", invalidPageArgs, errorFunc, nil},
		{"invalid height", invalidHeightArgs, errorFunc, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Validators(tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)
			if !tt.wantErr(t, err, fmt.Sprintf("Validators(%v, %v, %v, %v)",
				tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Validators(%v, %v, %v, %v)",
				tt.args.ctx, tt.args.heightPtr, tt.args.pagePtr, tt.args.perPagePtr)
		})
	}
}
