package proxy

import (
	tmabci "github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/line/ostracon/abci/client"
	abci "github.com/line/ostracon/abci/types"
)

//nolint
//go:generate mockery --case underscore --name AppConnConsensus|AppConnMempool|AppConnQuery|AppConnSnapshot|ClientCreator

//----------------------------------------------------------------------------------------
// Enforce which abci msgs can be sent on a connection at the type level

type AppConnConsensus interface {
	SetGlobalCallback(abcicli.GlobalCallback)
	Error() error

	InitChainSync(abci.RequestInitChain) (*abci.ResponseInitChain, error)

	BeginBlockSync(abci.RequestBeginBlock) (*tmabci.ResponseBeginBlock, error)
	DeliverTxAsync(tmabci.RequestDeliverTx, abcicli.ResponseCallback) *abcicli.ReqRes
	EndBlockSync(tmabci.RequestEndBlock) (*abci.ResponseEndBlock, error)
	CommitSync() (*tmabci.ResponseCommit, error)
}

type AppConnMempool interface {
	SetGlobalCallback(abcicli.GlobalCallback)
	Error() error

	CheckTxAsync(tmabci.RequestCheckTx, abcicli.ResponseCallback) *abcicli.ReqRes
	CheckTxSync(tmabci.RequestCheckTx) (*abci.ResponseCheckTx, error)

	BeginRecheckTxSync(abci.RequestBeginRecheckTx) (*abci.ResponseBeginRecheckTx, error)
	EndRecheckTxSync(abci.RequestEndRecheckTx) (*abci.ResponseEndRecheckTx, error)

	FlushAsync(abcicli.ResponseCallback) *abcicli.ReqRes
	FlushSync() (*tmabci.ResponseFlush, error)
}

type AppConnQuery interface {
	Error() error

	EchoSync(string) (*tmabci.ResponseEcho, error)
	InfoSync(tmabci.RequestInfo) (*tmabci.ResponseInfo, error)
	QuerySync(tmabci.RequestQuery) (*tmabci.ResponseQuery, error)

	//	SetOptionSync(key string, value string) (res abci.Result)
}

type AppConnSnapshot interface {
	Error() error

	ListSnapshotsSync(tmabci.RequestListSnapshots) (*tmabci.ResponseListSnapshots, error)
	OfferSnapshotSync(tmabci.RequestOfferSnapshot) (*tmabci.ResponseOfferSnapshot, error)
	LoadSnapshotChunkSync(tmabci.RequestLoadSnapshotChunk) (*tmabci.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkSync(tmabci.RequestApplySnapshotChunk) (*tmabci.ResponseApplySnapshotChunk, error)
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

func (app *appConnConsensus) BeginBlockSync(req abci.RequestBeginBlock) (*tmabci.ResponseBeginBlock, error) {
	return app.appConn.BeginBlockSync(req)
}

func (app *appConnConsensus) DeliverTxAsync(req tmabci.RequestDeliverTx, cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.DeliverTxAsync(req, cb)
}

func (app *appConnConsensus) EndBlockSync(req tmabci.RequestEndBlock) (*abci.ResponseEndBlock, error) {
	return app.appConn.EndBlockSync(req)
}

func (app *appConnConsensus) CommitSync() (*tmabci.ResponseCommit, error) {
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

func (app *appConnMempool) FlushSync() (*tmabci.ResponseFlush, error) {
	return app.appConn.FlushSync()
}

func (app *appConnMempool) CheckTxAsync(req tmabci.RequestCheckTx, cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.CheckTxAsync(req, cb)
}

func (app *appConnMempool) CheckTxSync(req tmabci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return app.appConn.CheckTxSync(req)
}

func (app *appConnMempool) BeginRecheckTxSync(req abci.RequestBeginRecheckTx) (*abci.ResponseBeginRecheckTx, error) {
	return app.appConn.BeginRecheckTxSync(req)
}

func (app *appConnMempool) EndRecheckTxSync(req abci.RequestEndRecheckTx) (*abci.ResponseEndRecheckTx, error) {
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

func (app *appConnQuery) EchoSync(msg string) (*tmabci.ResponseEcho, error) {
	return app.appConn.EchoSync(msg)
}

func (app *appConnQuery) InfoSync(req tmabci.RequestInfo) (*tmabci.ResponseInfo, error) {
	return app.appConn.InfoSync(req)
}

func (app *appConnQuery) QuerySync(reqQuery tmabci.RequestQuery) (*tmabci.ResponseQuery, error) {
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

func (app *appConnSnapshot) ListSnapshotsSync(req tmabci.RequestListSnapshots) (*tmabci.ResponseListSnapshots, error) {
	return app.appConn.ListSnapshotsSync(req)
}

func (app *appConnSnapshot) OfferSnapshotSync(req tmabci.RequestOfferSnapshot) (*tmabci.ResponseOfferSnapshot, error) {
	return app.appConn.OfferSnapshotSync(req)
}

func (app *appConnSnapshot) LoadSnapshotChunkSync(
	req tmabci.RequestLoadSnapshotChunk) (*tmabci.ResponseLoadSnapshotChunk, error) {
	return app.appConn.LoadSnapshotChunkSync(req)
}

func (app *appConnSnapshot) ApplySnapshotChunkSync(
	req tmabci.RequestApplySnapshotChunk) (*tmabci.ResponseApplySnapshotChunk, error) {
	return app.appConn.ApplySnapshotChunkSync(req)
}
