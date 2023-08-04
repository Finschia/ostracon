package proxy

import (
	"github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/Finschia/ostracon/abci/client"
	ocabci "github.com/Finschia/ostracon/abci/types"
)

//nolint
//go:generate ../scripts/mockery_generate.sh AppConnConsensus|AppConnMempool|AppConnQuery|AppConnSnapshot

//----------------------------------------------------------------------------------------
// Enforce which abci msgs can be sent on a connection at the type level

type AppConnConsensus interface {
	SetGlobalCallback(abcicli.GlobalCallback)
	Error() error

	InitChainSync(types.RequestInitChain) (*types.ResponseInitChain, error)

	BeginBlockSync(types.RequestBeginBlock) (*types.ResponseBeginBlock, error)
	DeliverTxAsync(types.RequestDeliverTx, abcicli.ResponseCallback) *abcicli.ReqRes
	EndBlockSync(types.RequestEndBlock) (*types.ResponseEndBlock, error)
	CommitSync() (*types.ResponseCommit, error)
}

type AppConnMempool interface {
	SetGlobalCallback(abcicli.GlobalCallback)
	Error() error

	CheckTxAsync(types.RequestCheckTx, abcicli.ResponseCallback) *abcicli.ReqRes
	CheckTxSync(types.RequestCheckTx) (*ocabci.ResponseCheckTx, error)

	BeginRecheckTxSync(ocabci.RequestBeginRecheckTx) (*ocabci.ResponseBeginRecheckTx, error)
	EndRecheckTxSync(ocabci.RequestEndRecheckTx) (*ocabci.ResponseEndRecheckTx, error)

	FlushAsync(abcicli.ResponseCallback) *abcicli.ReqRes
	FlushSync() (*types.ResponseFlush, error)
}

type AppConnQuery interface {
	Error() error

	EchoSync(string) (*types.ResponseEcho, error)
	InfoSync(types.RequestInfo) (*types.ResponseInfo, error)
	QuerySync(types.RequestQuery) (*types.ResponseQuery, error)

	//	SetOptionSync(key string, value string) (res ocabci.Result)
}

type AppConnSnapshot interface {
	Error() error

	ListSnapshotsSync(types.RequestListSnapshots) (*types.ResponseListSnapshots, error)
	OfferSnapshotSync(types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error)
	LoadSnapshotChunkSync(types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkSync(types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error)
}

//-----------------------------------------------------------------------------------------
// Implements AppConnConsensus (subset of abcicli.Client)

type appConnConsensus struct {
	appConn abcicli.Client
}

func NewAppConnConsensus(appConn abcicli.Client) AppConnConsensus {
	return &appConnConsensus{
		appConn: appConn,
	}
}

func (app *appConnConsensus) SetGlobalCallback(globalCb abcicli.GlobalCallback) {
	app.appConn.SetGlobalCallback(globalCb)
}

func (app *appConnConsensus) Error() error {
	return app.appConn.Error()
}

func (app *appConnConsensus) InitChainSync(req types.RequestInitChain) (*types.ResponseInitChain, error) {
	return app.appConn.InitChainSync(req)
}

func (app *appConnConsensus) BeginBlockSync(req types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	return app.appConn.BeginBlockSync(req)
}

func (app *appConnConsensus) DeliverTxAsync(req types.RequestDeliverTx, cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.DeliverTxAsync(req, cb)
}

func (app *appConnConsensus) EndBlockSync(req types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	return app.appConn.EndBlockSync(req)
}

func (app *appConnConsensus) CommitSync() (*types.ResponseCommit, error) {
	return app.appConn.CommitSync()
}

//------------------------------------------------
// Implements AppConnMempool (subset of abcicli.Client)

type appConnMempool struct {
	appConn abcicli.Client
}

func NewAppConnMempool(appConn abcicli.Client) AppConnMempool {
	return &appConnMempool{
		appConn: appConn,
	}
}

func (app *appConnMempool) SetGlobalCallback(globalCb abcicli.GlobalCallback) {
	app.appConn.SetGlobalCallback(globalCb)
}

func (app *appConnMempool) Error() error {
	return app.appConn.Error()
}

func (app *appConnMempool) FlushAsync(cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.FlushAsync(cb)
}

func (app *appConnMempool) FlushSync() (*types.ResponseFlush, error) {
	return app.appConn.FlushSync()
}

func (app *appConnMempool) CheckTxAsync(req types.RequestCheckTx, cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.CheckTxAsync(req, cb)
}

func (app *appConnMempool) CheckTxSync(req types.RequestCheckTx) (*ocabci.ResponseCheckTx, error) {
	return app.appConn.CheckTxSync(req)
}

func (app *appConnMempool) BeginRecheckTxSync(req ocabci.RequestBeginRecheckTx) (*ocabci.ResponseBeginRecheckTx, error) {
	return app.appConn.BeginRecheckTxSync(req)
}

func (app *appConnMempool) EndRecheckTxSync(req ocabci.RequestEndRecheckTx) (*ocabci.ResponseEndRecheckTx, error) {
	return app.appConn.EndRecheckTxSync(req)
}

//------------------------------------------------
// Implements AppConnQuery (subset of abcicli.Client)

type appConnQuery struct {
	appConn abcicli.Client
}

func NewAppConnQuery(appConn abcicli.Client) AppConnQuery {
	return &appConnQuery{
		appConn: appConn,
	}
}

func (app *appConnQuery) Error() error {
	return app.appConn.Error()
}

func (app *appConnQuery) EchoSync(msg string) (*types.ResponseEcho, error) {
	return app.appConn.EchoSync(msg)
}

func (app *appConnQuery) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	return app.appConn.InfoSync(req)
}

func (app *appConnQuery) QuerySync(reqQuery types.RequestQuery) (*types.ResponseQuery, error) {
	return app.appConn.QuerySync(reqQuery)
}

//------------------------------------------------
// Implements AppConnSnapshot (subset of abcicli.Client)

type appConnSnapshot struct {
	appConn abcicli.Client
}

func NewAppConnSnapshot(appConn abcicli.Client) AppConnSnapshot {
	return &appConnSnapshot{
		appConn: appConn,
	}
}

func (app *appConnSnapshot) Error() error {
	return app.appConn.Error()
}

func (app *appConnSnapshot) ListSnapshotsSync(req types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	return app.appConn.ListSnapshotsSync(req)
}

func (app *appConnSnapshot) OfferSnapshotSync(req types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	return app.appConn.OfferSnapshotSync(req)
}

func (app *appConnSnapshot) LoadSnapshotChunkSync(
	req types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	return app.appConn.LoadSnapshotChunkSync(req)
}

func (app *appConnSnapshot) ApplySnapshotChunkSync(
	req types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	return app.appConn.ApplySnapshotChunkSync(req)
}
