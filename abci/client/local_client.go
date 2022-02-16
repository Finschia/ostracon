package abcicli

import (
	types "github.com/line/ostracon/abci/types"
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
	types.Application

	globalCbMtx tmsync.Mutex
	globalCb    GlobalCallback
}

var _ Client = (*localClient)(nil)

// NewLocalClient creates a local client, which will be directly calling the
// methods of the given app.
//
// Both Async and Sync methods ignore the given context.Context parameter.
func NewLocalClient(mtx *tmsync.Mutex, app types.Application) Client {
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

// TODO: change types.Application to include Error()?
func (app *localClient) Error() error {
	return nil
}

func (app *localClient) FlushAsync(cb ResponseCallback) *ReqRes {
	// Do nothing
	reqRes := NewReqRes(types.ToRequestFlush(), cb)
	return app.done(reqRes, types.ToResponseFlush())
}

func (app *localClient) EchoAsync(msg string, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestEcho(msg), cb)
	return app.done(reqRes, types.ToResponseEcho(msg))
}

func (app *localClient) InfoAsync(req types.RequestInfo, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestInfo(req), cb)
	res := app.Application.Info(req)
	return app.done(reqRes, types.ToResponseInfo(res))
}

func (app *localClient) SetOptionAsync(req types.RequestSetOption, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestSetOption(req), cb)
	res := app.Application.SetOption(req)
	return app.done(reqRes, types.ToResponseSetOption(res))
}

func (app *localClient) DeliverTxAsync(req types.RequestDeliverTx, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestDeliverTx(req), cb)
	res := app.Application.DeliverTx(req)
	return app.done(reqRes, types.ToResponseDeliverTx(res))
}

func (app *localClient) CheckTxAsync(req types.RequestCheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestCheckTx(req), cb)

	app.Application.CheckTxAsync(req, func(r types.ResponseCheckTx) {
		res := types.ToResponseCheckTx(r)
		app.done(reqRes, res)
	})

	return reqRes
}

func (app *localClient) QueryAsync(req types.RequestQuery, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestQuery(req), cb)
	res := app.Application.Query(req)
	return app.done(reqRes, types.ToResponseQuery(res))
}

func (app *localClient) CommitAsync(cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestCommit(), cb)
	res := app.Application.Commit()
	return app.done(reqRes, types.ToResponseCommit(res))
}

func (app *localClient) InitChainAsync(req types.RequestInitChain, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestInitChain(req), cb)
	res := app.Application.InitChain(req)
	return app.done(reqRes, types.ToResponseInitChain(res))
}

func (app *localClient) BeginBlockAsync(req types.RequestBeginBlock, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestBeginBlock(req), cb)
	res := app.Application.BeginBlock(req)
	return app.done(reqRes, types.ToResponseBeginBlock(res))
}

func (app *localClient) EndBlockAsync(req types.RequestEndBlock, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestEndBlock(req), cb)
	res := app.Application.EndBlock(req)
	return app.done(reqRes, types.ToResponseEndBlock(res))
}

func (app *localClient) BeginRecheckTxAsync(req types.RequestBeginRecheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestBeginRecheckTx(req), cb)
	res := app.Application.BeginRecheckTx(req)
	return app.done(reqRes, types.ToResponseBeginRecheckTx(res))
}

func (app *localClient) EndRecheckTxAsync(req types.RequestEndRecheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestEndRecheckTx(req), cb)
	res := app.Application.EndRecheckTx(req)
	return app.done(reqRes, types.ToResponseEndRecheckTx(res))
}

func (app *localClient) ListSnapshotsAsync(req types.RequestListSnapshots, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestListSnapshots(req), cb)
	res := app.Application.ListSnapshots(req)
	return app.done(reqRes, types.ToResponseListSnapshots(res))
}

func (app *localClient) OfferSnapshotAsync(req types.RequestOfferSnapshot, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestOfferSnapshot(req), cb)
	res := app.Application.OfferSnapshot(req)
	return app.done(reqRes, types.ToResponseOfferSnapshot(res))
}

func (app *localClient) LoadSnapshotChunkAsync(req types.RequestLoadSnapshotChunk, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestLoadSnapshotChunk(req), cb)
	res := app.Application.LoadSnapshotChunk(req)
	return app.done(reqRes, types.ToResponseLoadSnapshotChunk(res))
}

func (app *localClient) ApplySnapshotChunkAsync(req types.RequestApplySnapshotChunk, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(types.ToRequestApplySnapshotChunk(req), cb)
	res := app.Application.ApplySnapshotChunk(req)
	return app.done(reqRes, types.ToResponseApplySnapshotChunk(res))
}

//-------------------------------------------------------
func (app *localClient) FlushSync() (*types.ResponseFlush, error) {
	return &types.ResponseFlush{}, nil
}

func (app *localClient) EchoSync(msg string) (*types.ResponseEcho, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	return &types.ResponseEcho{Message: msg}, nil
}

func (app *localClient) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Info(req)
	return &res, nil
}

func (app *localClient) SetOptionSync(req types.RequestSetOption) (*types.ResponseSetOption, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.SetOption(req)
	return &res, nil
}

func (app *localClient) DeliverTxSync(req types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.DeliverTx(req)
	return &res, nil
}

func (app *localClient) CheckTxSync(req types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.CheckTxSync(req)
	return &res, nil
}

func (app *localClient) QuerySync(req types.RequestQuery) (*types.ResponseQuery, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Query(req)
	return &res, nil
}

func (app *localClient) CommitSync() (*types.ResponseCommit, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Commit()
	return &res, nil
}

func (app *localClient) InitChainSync(req types.RequestInitChain) (*types.ResponseInitChain, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.InitChain(req)
	return &res, nil
}

func (app *localClient) BeginBlockSync(req types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.BeginBlock(req)
	return &res, nil
}

func (app *localClient) EndBlockSync(req types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.EndBlock(req)
	return &res, nil
}

func (app *localClient) BeginRecheckTxSync(req types.RequestBeginRecheckTx) (*types.ResponseBeginRecheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.BeginRecheckTx(req)
	return &res, nil
}

func (app *localClient) EndRecheckTxSync(req types.RequestEndRecheckTx) (*types.ResponseEndRecheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.EndRecheckTx(req)
	return &res, nil
}

func (app *localClient) ListSnapshotsSync(req types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.ListSnapshots(req)
	return &res, nil
}

func (app *localClient) OfferSnapshotSync(req types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.OfferSnapshot(req)
	return &res, nil
}

func (app *localClient) LoadSnapshotChunkSync(
	req types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.LoadSnapshotChunk(req)
	return &res, nil
}

func (app *localClient) ApplySnapshotChunkSync(
	req types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.ApplySnapshotChunk(req)
	return &res, nil
}

//-------------------------------------------------------

func (app *localClient) done(reqRes *ReqRes, res *types.Response) *ReqRes {
	set := reqRes.SetDone(res)
	if set {
		if globalCb := app.GetGlobalCallback(); globalCb != nil {
			globalCb(reqRes.Request, res)
		}
	}
	return reqRes
}
