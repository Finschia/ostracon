package abcicli

import (
	tmabci "github.com/tendermint/tendermint/abci/types"

	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/service"
	tmsync "github.com/line/ostracon/libs/sync"
)

var _ Client = (*localClient)(nil)

// NOTE: use defer to unlock mutex because Application might panic (e.g., in
// case of malicious tx or query). It only makes sense for publicly exposed
// methods like CheckTx (/broadcast_tx_* RPC endpoint) or Query (/abci_query
// RPC endpoint), but defers are used everywhere for the sake of consistency.
type localClient struct {
	service.BaseService

	// TODO: remove `mtx` to increase concurrency. We could remove it because the app should protect itself.
	mtx *tmsync.Mutex
	// CONTRACT: The application should protect itself from concurrency as an abci server.
	abci.Application

	globalCbMtx tmsync.Mutex
	globalCb    GlobalCallback
}

var _ Client = (*localClient)(nil)

// NewLocalClient creates a local client, which will be directly calling the
// methods of the given app.
//
// Both Async and Sync methods ignore the given context.Context parameter.
func NewLocalClient(mtx *tmsync.Mutex, app abci.Application) Client {
	if mtx == nil {
		mtx = new(tmsync.Mutex)
	}
	cli := &localClient{
		mtx:         mtx,
		Application: app,
	}
	cli.BaseService = *service.NewBaseService(nil, "localClient", cli)
	return cli
}

func (app *localClient) SetGlobalCallback(globalCb GlobalCallback) {
	app.globalCbMtx.Lock()
	defer app.globalCbMtx.Unlock()
	app.globalCb = globalCb
}

func (app *localClient) GetGlobalCallback() (cb GlobalCallback) {
	app.globalCbMtx.Lock()
	defer app.globalCbMtx.Unlock()
	cb = app.globalCb
	return cb
}

// TODO: change abci.Application to include Error()?
func (app *localClient) Error() error {
	return nil
}

func (app *localClient) FlushAsync(cb ResponseCallback) *ReqRes {
	// Do nothing
	reqRes := NewReqRes(abci.ToRequestFlush(), cb)
	return app.done(reqRes, abci.ToResponseFlush())
}

func (app *localClient) EchoAsync(msg string, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestEcho(msg), cb)
	return app.done(reqRes, abci.ToResponseEcho(msg))
}

func (app *localClient) InfoAsync(req tmabci.RequestInfo, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestInfo(req), cb)
	res := app.Application.Info(req)
	return app.done(reqRes, abci.ToResponseInfo(res))
}

func (app *localClient) SetOptionAsync(req tmabci.RequestSetOption, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestSetOption(req), cb)
	res := app.Application.SetOption(req)
	return app.done(reqRes, abci.ToResponseSetOption(res))
}

func (app *localClient) DeliverTxAsync(req tmabci.RequestDeliverTx, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestDeliverTx(req), cb)
	res := app.Application.DeliverTx(req)
	return app.done(reqRes, abci.ToResponseDeliverTx(res))
}

func (app *localClient) CheckTxAsync(req tmabci.RequestCheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestCheckTx(req), cb)

	app.Application.CheckTxAsync(req, func(r abci.ResponseCheckTx) {
		res := abci.ToResponseCheckTx(r)
		app.done(reqRes, res)
	})

	return reqRes
}

func (app *localClient) QueryAsync(req tmabci.RequestQuery, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestQuery(req), cb)
	res := app.Application.Query(req)
	return app.done(reqRes, abci.ToResponseQuery(res))
}

func (app *localClient) CommitAsync(cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestCommit(), cb)
	res := app.Application.Commit()
	return app.done(reqRes, abci.ToResponseCommit(res))
}

func (app *localClient) InitChainAsync(req abci.RequestInitChain, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestInitChain(req), cb)
	res := app.Application.InitChain(req)
	return app.done(reqRes, abci.ToResponseInitChain(res))
}

func (app *localClient) BeginBlockAsync(req abci.RequestBeginBlock, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestBeginBlock(req), cb)
	res := app.Application.BeginBlock(req)
	return app.done(reqRes, abci.ToResponseBeginBlock(res))
}

func (app *localClient) EndBlockAsync(req tmabci.RequestEndBlock, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestEndBlock(req), cb)
	res := app.Application.EndBlock(req)
	return app.done(reqRes, abci.ToResponseEndBlock(res))
}

func (app *localClient) BeginRecheckTxAsync(req abci.RequestBeginRecheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestBeginRecheckTx(req), cb)
	res := app.Application.BeginRecheckTx(req)
	return app.done(reqRes, abci.ToResponseBeginRecheckTx(res))
}

func (app *localClient) EndRecheckTxAsync(req abci.RequestEndRecheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestEndRecheckTx(req), cb)
	res := app.Application.EndRecheckTx(req)
	return app.done(reqRes, abci.ToResponseEndRecheckTx(res))
}

func (app *localClient) ListSnapshotsAsync(req tmabci.RequestListSnapshots, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestListSnapshots(req), cb)
	res := app.Application.ListSnapshots(req)
	return app.done(reqRes, abci.ToResponseListSnapshots(res))
}

func (app *localClient) OfferSnapshotAsync(req tmabci.RequestOfferSnapshot, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestOfferSnapshot(req), cb)
	res := app.Application.OfferSnapshot(req)
	return app.done(reqRes, abci.ToResponseOfferSnapshot(res))
}

func (app *localClient) LoadSnapshotChunkAsync(req tmabci.RequestLoadSnapshotChunk, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestLoadSnapshotChunk(req), cb)
	res := app.Application.LoadSnapshotChunk(req)
	return app.done(reqRes, abci.ToResponseLoadSnapshotChunk(res))
}

func (app *localClient) ApplySnapshotChunkAsync(req tmabci.RequestApplySnapshotChunk, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(abci.ToRequestApplySnapshotChunk(req), cb)
	res := app.Application.ApplySnapshotChunk(req)
	return app.done(reqRes, abci.ToResponseApplySnapshotChunk(res))
}

//-------------------------------------------------------
func (app *localClient) FlushSync() (*tmabci.ResponseFlush, error) {
	return &tmabci.ResponseFlush{}, nil
}

func (app *localClient) EchoSync(msg string) (*tmabci.ResponseEcho, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	return &tmabci.ResponseEcho{Message: msg}, nil
}

func (app *localClient) InfoSync(req tmabci.RequestInfo) (*tmabci.ResponseInfo, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Info(req)
	return &res, nil
}

func (app *localClient) SetOptionSync(req tmabci.RequestSetOption) (*tmabci.ResponseSetOption, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.SetOption(req)
	return &res, nil
}

func (app *localClient) DeliverTxSync(req tmabci.RequestDeliverTx) (*tmabci.ResponseDeliverTx, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.DeliverTx(req)
	return &res, nil
}

func (app *localClient) CheckTxSync(req tmabci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.CheckTxSync(req)
	return &res, nil
}

func (app *localClient) QuerySync(req tmabci.RequestQuery) (*tmabci.ResponseQuery, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Query(req)
	return &res, nil
}

func (app *localClient) CommitSync() (*tmabci.ResponseCommit, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Commit()
	return &res, nil
}

func (app *localClient) InitChainSync(req abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.InitChain(req)
	return &res, nil
}

func (app *localClient) BeginBlockSync(req abci.RequestBeginBlock) (*tmabci.ResponseBeginBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.BeginBlock(req)
	return &res, nil
}

func (app *localClient) EndBlockSync(req tmabci.RequestEndBlock) (*abci.ResponseEndBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.EndBlock(req)
	return &res, nil
}

func (app *localClient) BeginRecheckTxSync(req abci.RequestBeginRecheckTx) (*abci.ResponseBeginRecheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.BeginRecheckTx(req)
	return &res, nil
}

func (app *localClient) EndRecheckTxSync(req abci.RequestEndRecheckTx) (*abci.ResponseEndRecheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.EndRecheckTx(req)
	return &res, nil
}

func (app *localClient) ListSnapshotsSync(req tmabci.RequestListSnapshots) (*tmabci.ResponseListSnapshots, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.ListSnapshots(req)
	return &res, nil
}

func (app *localClient) OfferSnapshotSync(req tmabci.RequestOfferSnapshot) (*tmabci.ResponseOfferSnapshot, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.OfferSnapshot(req)
	return &res, nil
}

func (app *localClient) LoadSnapshotChunkSync(
	req tmabci.RequestLoadSnapshotChunk) (*tmabci.ResponseLoadSnapshotChunk, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.LoadSnapshotChunk(req)
	return &res, nil
}

func (app *localClient) ApplySnapshotChunkSync(
	req tmabci.RequestApplySnapshotChunk) (*tmabci.ResponseApplySnapshotChunk, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.ApplySnapshotChunk(req)
	return &res, nil
}

//-------------------------------------------------------

func (app *localClient) done(reqRes *ReqRes, res *abci.Response) *ReqRes {
	set := reqRes.SetDone(res)
	if set {
		if globalCb := app.GetGlobalCallback(); globalCb != nil {
			globalCb(reqRes.Request, res)
		}
	}
	return reqRes
}
