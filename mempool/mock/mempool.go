package mock

import (
	abci "github.com/tendermint/tendermint/abci/types"

	ocabci "github.com/Finschia/ostracon/abci/types"
	"github.com/Finschia/ostracon/libs/clist"
	"github.com/Finschia/ostracon/mempool"
	"github.com/Finschia/ostracon/types"
)

// Mempool is an empty implementation of a Mempool, useful for testing.
type Mempool struct{}

var _ mempool.Mempool = Mempool{}

func (Mempool) Lock()     {}
func (Mempool) Unlock()   {}
func (Mempool) Size() int { return 0 }
func (Mempool) CheckTxSync(_ types.Tx, _ func(*ocabci.Response), _ mempool.TxInfo) error {
	return nil
}
func (Mempool) CheckTxAsync(_ types.Tx, _ mempool.TxInfo, _ func(error), _ func(*ocabci.Response)) {
}
func (Mempool) RemoveTxByKey(txKey types.TxKey) error            { return nil }
func (Mempool) ReapMaxBytesMaxGas(_, _ int64) types.Txs          { return types.Txs{} }
func (Mempool) ReapMaxBytesMaxGasMaxTxs(_, _, _ int64) types.Txs { return types.Txs{} }
func (Mempool) ReapMaxTxs(n int) types.Txs                       { return types.Txs{} }
func (Mempool) Update(
	_ *types.Block,
	_ []*abci.ResponseDeliverTx,
	_ mempool.PreCheckFunc,
	_ mempool.PostCheckFunc,
) error {
	return nil
}
func (Mempool) Flush()                        {}
func (Mempool) FlushAppConn() error           { return nil }
func (Mempool) TxsAvailable() <-chan struct{} { return make(chan struct{}) }
func (Mempool) EnableTxsAvailable()           {}
func (Mempool) SizeBytes() int64              { return 0 }

func (Mempool) TxsFront() *clist.CElement    { return nil }
func (Mempool) TxsWaitChan() <-chan struct{} { return nil }

func (Mempool) InitWAL() error { return nil }
func (Mempool) CloseWAL()      {}
