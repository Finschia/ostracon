package proxy

import (
	abci "github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/line/ostracon/abci/client"
	ocabci "github.com/line/ostracon/abci/types"
)

//nolint
//go:generate mockery --case underscore --name AppConnConsensus|AppConnMempool|AppConnQuery|AppConnSnapshot|ClientCreator

//----------------------------------------------------------------------------------------
// Enforce which abci msgs can be sent on a connection at the type level

type AppConnConsensus interface {
	SetGlobalCallback(abcicli.GlobalCallback)
	Error() error

	InitChainSync(abci.RequestInitChain) (*abci.ResponseInitChain, error)

	BeginBlockSync(ocabci.RequestBeginBlock) (*abci.ResponseBeginBlock, error)
	DeliverTxAsync(abci.RequestDeliverTx, abcicli.ResponseCallback) *abcicli.ReqRes
	EndBlockSync(abci.RequestEndBlock) (*abci.ResponseEndBlock, error)
	CommitSync() (*abci.ResponseCommit, error)
}

type AppConnMempool interface {
	SetGlobalCallback(abcicli.GlobalCallback)
	Error() error

	CheckTxAsync(abci.RequestCheckTx, abcicli.ResponseCallback) *abcicli.ReqRes
	CheckTxSync(abci.RequestCheckTx) (*ocabci.ResponseCheckTx, error)

	BeginRecheckTxSync(ocabci.RequestBeginRecheckTx) (*ocabci.ResponseBeginRecheckTx, error)
	EndRecheckTxSync(ocabci.RequestEndRecheckTx) (*ocabci.ResponseEndRecheckTx, error)

	FlushAsync(abcicli.ResponseCallback) *abcicli.ReqRes
	FlushSync() (*abci.ResponseFlush, error)
}

type AppConnQuery interface {
	Error() error

	EchoSync(string) (*abci.ResponseEcho, error)
	InfoSync(abci.RequestInfo) (*abci.ResponseInfo, error)
	QuerySync(abci.RequestQuery) (*abci.ResponseQuery, error)

	//	SetOptionSync(key string, value string) (res ocabci.Result)
}

type AppConnSnapshot interface {
	Error() error

	ListSnapshotsSync(abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error)
	OfferSnapshotSync(abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error)
	LoadSnapshotChunkSync(abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error)
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

func (app *appConnConsensus) InitChainSync(req abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	return app.appConn.InitChainSync(req)
}

func (app *appConnConsensus) BeginBlockSync(req ocabci.RequestBeginBlock) (*abci.ResponseBeginBlock, error) {
	return app.appConn.BeginBlockSync(req)
}

func (app *appConnConsensus) DeliverTxAsync(req abci.RequestDeliverTx, cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.DeliverTxAsync(req, cb)
}

func (app *appConnConsensus) EndBlockSync(req abci.RequestEndBlock) (*abci.ResponseEndBlock, error) {
	return app.appConn.EndBlockSync(req)
}

func (app *appConnConsensus) CommitSync() (*abci.ResponseCommit, error) {
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

func (app *appConnMempool) FlushSync() (*abci.ResponseFlush, error) {
	return app.appConn.FlushSync()
}

func (app *appConnMempool) CheckTxAsync(req abci.RequestCheckTx, cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.CheckTxAsync(req, cb)
}

func (app *appConnMempool) CheckTxSync(req abci.RequestCheckTx) (*ocabci.ResponseCheckTx, error) {
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

func (app *appConnQuery) EchoSync(msg string) (*abci.ResponseEcho, error) {
	return app.appConn.EchoSync(msg)
}

func (app *appConnQuery) InfoSync(req abci.RequestInfo) (*abci.ResponseInfo, error) {
	return app.appConn.InfoSync(req)
}

func (app *appConnQuery) QuerySync(reqQuery abci.RequestQuery) (*abci.ResponseQuery, error) {
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

func (app *appConnSnapshot) ListSnapshotsSync(req abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return app.appConn.ListSnapshotsSync(req)
}

func (app *appConnSnapshot) OfferSnapshotSync(req abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return app.appConn.OfferSnapshotSync(req)
}

func (app *appConnSnapshot) LoadSnapshotChunkSync(
	req abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return app.appConn.LoadSnapshotChunkSync(req)
}

func (app *appConnSnapshot) ApplySnapshotChunkSync(
	req abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return app.appConn.ApplySnapshotChunkSync(req)
}
