package mock

import (
	abci "github.com/tendermint/tendermint/abci/types"

	ocabci "github.com/Finschia/ostracon/abci/types"
	"github.com/Finschia/ostracon/libs/clist"
	mempl "github.com/Finschia/ostracon/mempool"
	"github.com/Finschia/ostracon/types"
)

// Mempool is an empty implementation of a Mempool, useful for testing.
type Mempool struct{}

var _ mempl.Mempool = Mempool{}

func (Mempool) Lock()     {}
func (Mempool) Unlock()   {}
func (Mempool) Size() int { return 0 }
func (Mempool) CheckTxSync(_ types.Tx, _ func(*ocabci.Response), _ mempl.TxInfo) error {
	return nil
}
func (Mempool) CheckTxAsync(_ types.Tx, _ mempl.TxInfo, _ func(error), _ func(*ocabci.Response)) {
}
func (Mempool) ReapMaxBytesMaxGas(_, _ int64) types.Txs          { return types.Txs{} }
func (Mempool) ReapMaxBytesMaxGasMaxTxs(_, _, _ int64) types.Txs { return types.Txs{} }
func (Mempool) ReapMaxTxs(n int) types.Txs                       { return types.Txs{} }
func (Mempool) Update(
	_ *types.Block,
	_ []*abci.ResponseDeliverTx,
	_ mempl.PreCheckFunc,
	_ mempl.PostCheckFunc,
) error {
	return nil
}
func (Mempool) Flush()                        {}
func (Mempool) FlushAppConn() error           { return nil }
func (Mempool) TxsAvailable() <-chan struct{} { return make(chan struct{}) }
func (Mempool) EnableTxsAvailable()           {}
func (Mempool) TxsBytes() int64               { return 0 }

func (Mempool) TxsFront() *clist.CElement    { return nil }
func (Mempool) TxsWaitChan() <-chan struct{} { return nil }

func (Mempool) InitWAL() error { return nil }
func (Mempool) CloseWAL()      {}
