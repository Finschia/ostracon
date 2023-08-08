package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	ocabcicli "github.com/Finschia/ostracon/abci/client"
	"github.com/Finschia/ostracon/config"
	"github.com/Finschia/ostracon/libs/log"
	"github.com/Finschia/ostracon/mempool"
	memv0 "github.com/Finschia/ostracon/mempool/v0"
	"github.com/Finschia/ostracon/proxy/mocks"
	ctypes "github.com/Finschia/ostracon/rpc/core/types"
	rpctypes "github.com/Finschia/ostracon/rpc/jsonrpc/types"
	"github.com/Finschia/ostracon/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	abci "github.com/tendermint/tendermint/abci/types"
)

type ErrorAssertionFunc func(t assert.TestingT, theError error, contains string, msgAndArgs ...interface{}) bool

func TestBroadcastTxAsync(t *testing.T) {
	type args struct {
		ctx *rpctypes.Context
		tx  types.Tx
	}
	tx := types.Tx{}
	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultBroadcastTx
		wantErr assert.ErrorAssertionFunc
		err     error
	}{
		{
			name: "success",
			args: args{
				ctx: &rpctypes.Context{},
				tx:  tx,
			},
			want: &ctypes.ResultBroadcastTx{
				Code:         abci.CodeTypeOK,
				Data:         nil,
				Log:          "",
				Codespace:    "",
				MempoolError: "",
				Hash:         tx.Hash(),
			},
			wantErr: assert.NoError,
			err:     nil,
		},
		{
			name: "failure: tx is same the one before",
			args: args{
				ctx: &rpctypes.Context{},
				tx:  tx,
			},
			want:    nil,
			wantErr: assert.Error,
			err:     mempool.ErrTxInCache,
		},
	}
	env = &Environment{}
	mockAppConnMempool := &mocks.AppConnMempool{}
	mockAppConnMempool.On("SetGlobalCallback", mock.Anything)
	mockAppConnMempool.On("CheckTxAsync", mock.Anything, mock.Anything).Return(&ocabcicli.ReqRes{})
	env.Mempool = memv0.NewCListMempool(config.TestConfig().Mempool, mockAppConnMempool, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAppConnMempool.On("Error").Return(tt.err).Once()
			got, err := BroadcastTxAsync(tt.args.ctx, tt.args.tx)
			if !tt.wantErr(t, err, fmt.Sprintf("BroadcastTxAsync(%v, %v)", tt.args.ctx, tt.args.tx)) {
				return
			}
			assert.Equal(t, tt.err, err)
			assert.Equalf(t, tt.want, got, "BroadcastTxAsync(%v, %v)", tt.args.ctx, tt.args.tx)
		})
	}
}

func TestBroadcastTxSync(t *testing.T) {
	type args struct {
		ctx *rpctypes.Context
		tx  types.Tx
	}
	tx := types.Tx{}
	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultBroadcastTx
		wantErr assert.ErrorAssertionFunc
		err     error
	}{
		{
			name: "success",
			args: args{
				ctx: &rpctypes.Context{},
				tx:  tx,
			},
			want: &ctypes.ResultBroadcastTx{
				Code:         abci.CodeTypeOK,
				Data:         nil,
				Log:          "",
				Codespace:    "",
				MempoolError: "",
				Hash:         tx.Hash(),
			},
			wantErr: assert.NoError,
			err:     nil,
		},
		{
			name: "failure: tx is same the one before",
			args: args{
				ctx: &rpctypes.Context{},
				tx:  tx,
			},
			want:    nil,
			wantErr: assert.Error,
			err:     mempool.ErrTxInMap,
		},
	}
	env = &Environment{}
	mockAppConnMempool := &mocks.AppConnMempool{}
	mockAppConnMempool.On("SetGlobalCallback", mock.Anything)
	mockAppConnMempool.On("CheckTxSync", mock.Anything).Return(&abci.ResponseCheckTx{}, nil)
	env.Mempool = memv0.NewCListMempool(config.TestConfig().Mempool, mockAppConnMempool, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAppConnMempool.On("Error").Return(tt.err).Once()
			got, err := BroadcastTxSync(tt.args.ctx, tt.args.tx)
			if !tt.wantErr(t, err, fmt.Sprintf("BroadcastTxSync(%v, %v)", tt.args.ctx, tt.args.tx)) {
				return
			}
			assert.Equal(t, tt.err, err)
			assert.Equalf(t, tt.want, got, "BroadcastTxSync(%v, %v)", tt.args.ctx, tt.args.tx)
		})
	}
}

// TestBroadcastTxSyncWithCancelContextForCheckTxSync test in isolation from TestBroadcastTxSync since avoiding coexistence
func TestBroadcastTxSyncWithCancelContextForCheckTxSync(t *testing.T) {
	type args struct {
		ctx *rpctypes.Context
		tx  types.Tx
	}
	errContext, cancel := context.WithCancel(context.Background())
	defer cancel() // for safety to avoid memory leaks
	req := &http.Request{}
	req = req.WithContext(errContext)
	errRpcContext := rpctypes.Context{HTTPReq: req}
	tx := types.Tx{}
	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultBroadcastTx
		wantErr ErrorAssertionFunc
		err     error
	}{
		{
			name: "failure(non-deterministic test, retry please): interrupted by context",
			args: args{
				ctx: &errRpcContext,
				tx:  tx,
			},
			want:    nil,
			wantErr: assert.ErrorContains,
			err:     fmt.Errorf("broadcast confirmation not received: context canceled"),
		},
	}
	env = &Environment{}
	mockAppConnMempool := &mocks.AppConnMempool{}
	mockAppConnMempool.On("SetGlobalCallback", mock.Anything)
	mockAppConnMempool.On("CheckTxSync", mock.Anything).Return(
		&abci.ResponseCheckTx{Code: abci.CodeTypeOK}, nil).WaitUntil(
		time.After(1000 * time.Millisecond)) // Wait calling the context cancel
	mockAppConnMempool.On("Error").Return(nil) // Not to use tt.err
	env.Mempool = memv0.NewCListMempool(config.TestConfig().Mempool, mockAppConnMempool, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// cancel context for while doing `env.Mempool.CheckTxSync`
			cancel()
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				got, err := BroadcastTxSync(tt.args.ctx, tt.args.tx)
				if !tt.wantErr(t, err, tt.err.Error(), fmt.Sprintf("BroadcastTxSync(%v, %v)", tt.args.ctx, tt.args.tx)) {
					wg.Done()
					return
				}
				assert.Equalf(t, tt.want, got, "BroadcastTxSync(%v, %v)", tt.args.ctx, tt.args.tx)
				wg.Done()
			}()
			wg.Wait()
		})
	}
}

func TestBroadcastTxCommit(t *testing.T) {
	type args struct {
		ctx *rpctypes.Context
		tx  types.Tx
	}
	height := int64(1)
	tx := types.Tx{}
	tx1 := types.Tx{1}
	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultBroadcastTxCommit
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				ctx: &rpctypes.Context{},
				tx:  tx,
			},
			want: &ctypes.ResultBroadcastTxCommit{
				CheckTx: abci.ResponseCheckTx{
					Code: abci.CodeTypeOK,
				},
				DeliverTx: abci.ResponseDeliverTx{
					Code: abci.CodeTypeOK,
					Data: tx,
				},
				Hash:   tx.Hash(),
				Height: height,
			},
			wantErr: assert.NoError,
		},
		{
			name: "success but CheckTxResponse is not OK",
			args: args{
				ctx: &rpctypes.Context{},
				tx:  tx1,
			},
			want: &ctypes.ResultBroadcastTxCommit{
				CheckTx: abci.ResponseCheckTx{
					Code: abci.CodeTypeOK + 1, // Not OK
				},
				DeliverTx: abci.ResponseDeliverTx{}, // return empty response
				Hash:      tx1.Hash(),
				Height:    0, // return empty height
			},
			wantErr: assert.NoError,
		},
	}
	env = &Environment{}
	env.Logger = log.TestingLogger()
	env.Config = *config.TestConfig().RPC
	env.EventBus = types.NewEventBus()
	err := env.EventBus.OnStart()
	defer env.EventBus.OnStop()
	assert.NoError(t, err)
	mockAppConnMempool := &mocks.AppConnMempool{}
	mockAppConnMempool.On("SetGlobalCallback", mock.Anything)
	mockAppConnMempool.On("Error").Return(nil)
	env.Mempool = memv0.NewCListMempool(config.TestConfig().Mempool, mockAppConnMempool, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAppConnMempool.On("CheckTxSync", mock.Anything).Return(&tt.want.CheckTx, nil).Once()
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				got, err := BroadcastTxCommit(tt.args.ctx, tt.args.tx)
				if !tt.wantErr(t, err, fmt.Sprintf("BroadcastTxCommit(%v, %v)", tt.args.ctx, tt.args.tx)) {
					wg.Done()
					return
				}
				assert.Equalf(t, tt.want, got, "BroadcastTxCommit(%v, %v)", tt.args.ctx, tt.args.tx)
				wg.Done()
			}()
			// Wait the time for `env.EventBus.Subscribe` in BroadcastTxCommit
			time.Sleep(10 * time.Millisecond)
			err := env.EventBus.PublishEventTx(types.EventDataTx{
				TxResult: abci.TxResult{
					Height: height,
					Index:  0,
					Tx:     tt.args.tx,
					Result: tt.want.DeliverTx,
				},
			})
			assert.NoError(t, err)
			wg.Wait()
		})
	}
}

func TestBroadcastTxCommitWithCancelContextForCheckTxSync(t *testing.T) {
	type args struct {
		ctx *rpctypes.Context
		tx  types.Tx
	}
	errContext, cancel := context.WithCancel(context.Background())
	defer cancel() // for safety to avoid memory leaks
	req := &http.Request{}
	req = req.WithContext(errContext)
	errRpcContext := rpctypes.Context{HTTPReq: req}
	height := int64(1)
	tx := types.Tx{}
	resCheckTx := abci.ResponseCheckTx{
		Code: abci.CodeTypeOK,
	}
	resDeliverTx := abci.ResponseDeliverTx{
		Code: abci.CodeTypeOK,
		Data: tx,
	}
	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultBroadcastTxCommit
		wantErr ErrorAssertionFunc
		err     error
	}{
		{
			name: "failure(non-deterministic test, retry please): interrupted by context",
			args: args{
				ctx: &errRpcContext,
				tx:  tx,
			},
			want:    nil,
			wantErr: assert.ErrorContains,
			err:     fmt.Errorf("broadcast confirmation not received: context canceled"),
		},
	}
	env = &Environment{}
	env.Logger = log.TestingLogger()
	env.Config = *config.TestConfig().RPC
	env.EventBus = types.NewEventBus()
	err := env.EventBus.OnStart()
	defer env.EventBus.OnStop()
	assert.NoError(t, err)
	mockAppConnMempool := &mocks.AppConnMempool{}
	mockAppConnMempool.On("SetGlobalCallback", mock.Anything)
	mockAppConnMempool.On("Error").Return(nil)
	mockAppConnMempool.On("CheckTxSync", mock.Anything).Return(&resCheckTx, nil).WaitUntil(
		time.After(1 * time.Second)) // Wait calling the context cancel
	env.Mempool = memv0.NewCListMempool(config.TestConfig().Mempool, mockAppConnMempool, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			wg.Add(2)
			go func() {
				// Wait the time for `env.EventBus.Subscribe` in BroadcastTxCommit
				time.Sleep(10 * time.Millisecond)
				err := env.EventBus.PublishEventTx(types.EventDataTx{
					TxResult: abci.TxResult{
						Height: height,
						Index:  0,
						Tx:     tx,
						Result: resDeliverTx,
					},
				})
				assert.NoError(t, err)
				// cancel context for while doing `env.Mempool.CheckTxSync`
				cancel()
				wg.Done()
			}()
			go func() {
				got, err := BroadcastTxCommit(tt.args.ctx, tt.args.tx)
				if !tt.wantErr(t, err, tt.err.Error(), fmt.Sprintf("BroadcastTxCommit(%v, %v)", tt.args.ctx, tt.args.tx)) {
					wg.Done()
					return
				}
				assert.Equalf(t, tt.want, got, "BroadcastTxCommit(%v, %v)", tt.args.ctx, tt.args.tx)
				wg.Done()
			}()
			wg.Wait()
		})
	}
}

func TestBroadcastTxCommitTimeout(t *testing.T) {
	type args struct {
		ctx *rpctypes.Context
		tx  types.Tx
	}
	tx := types.Tx{}
	tests := []struct {
		name    string
		args    args
		want    *ctypes.ResultBroadcastTxCommit
		wantErr ErrorAssertionFunc
		err     error
	}{
		{
			name: "failure: timeout",
			args: args{
				ctx: &rpctypes.Context{},
				tx:  tx,
			},
			want: &ctypes.ResultBroadcastTxCommit{
				CheckTx: abci.ResponseCheckTx{
					Code: abci.CodeTypeOK,
				},
				DeliverTx: abci.ResponseDeliverTx{}, // return empty response
				Hash:      tx.Hash(),
				Height:    0, // return empty height
			},
			wantErr: assert.ErrorContains,
			err:     errors.New("timed out waiting for tx to be included in a block"),
		},
	}
	env = &Environment{}
	env.Logger = log.TestingLogger()
	env.Config = *config.TestConfig().RPC
	env.Config.TimeoutBroadcastTxCommit = 1 // For test
	env.EventBus = types.NewEventBus()
	err := env.EventBus.OnStart()
	defer env.EventBus.OnStop()
	assert.NoError(t, err)
	mockAppConnMempool := &mocks.AppConnMempool{}
	mockAppConnMempool.On("SetGlobalCallback", mock.Anything)
	mockAppConnMempool.On("Error").Return(nil)
	env.Mempool = memv0.NewCListMempool(config.TestConfig().Mempool, mockAppConnMempool, 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAppConnMempool.On("CheckTxSync", mock.Anything).Return(&tt.want.CheckTx, nil).Once()
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				got, err := BroadcastTxCommit(tt.args.ctx, tt.args.tx)
				if !tt.wantErr(t, err, tt.err.Error(), fmt.Sprintf("BroadcastTxCommit(%v, %v)", tt.args.ctx, tt.args.tx)) {
					wg.Done()
					return
				}
				assert.Equalf(t, tt.want, got, "BroadcastTxCommit(%v, %v)", tt.args.ctx, tt.args.tx)
				wg.Done()
			}()
			wg.Wait()
		})
	}
}
