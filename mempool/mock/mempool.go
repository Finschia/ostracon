package mock

import (
	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/clist"
	mempl "github.com/line/ostracon/mempool"
	"github.com/line/ostracon/types"
)

// Mempool is an empty implementation of a Mempool, useful for testing.
type Mempool struct{}

var _ mempl.Mempool = Mempool{}

func (Mempool) Lock()     {}
func (Mempool) Unlock()   {}
func (Mempool) Size() int { return 0 }
func (Mempool) CheckTxSync(_ types.Tx, _ mempl.TxInfo) (*abci.Response, error) {
	return nil, nil
}
func (Mempool) CheckTxAsync(_ types.Tx, _ mempl.TxInfo, _ func(*abci.Response)) error {
	return nil
}
func (Mempool) ReapMaxBytesMaxGas(_, _ int64) types.Txs { return types.Txs{} }
func (Mempool) ReapMaxTxs(n int) types.Txs              { return types.Txs{} }
func (Mempool) Update(
	_ *types.Block,
	_ []*abci.ResponseDeliverTx,
	_ mempl.PreCheckFunc,
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
