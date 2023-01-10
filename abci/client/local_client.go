package abcicli

import (
	abci "github.com/tendermint/tendermint/abci/types"

	ocabci "github.com/line/ostracon/abci/types"
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
	// CONTRACT: The application should protect itself from concurrency as an ocabci server.
	ocabci.Application

	globalCbMtx tmsync.Mutex
	globalCb    GlobalCallback
}

var _ Client = (*localClient)(nil)

// NewLocalClient creates a local client, which will be directly calling the
// methods of the given app.
//
// Both Async and Sync methods ignore the given context.Context parameter.
func NewLocalClient(mtx *tmsync.Mutex, app ocabci.Application) Client {
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
	reqRes := NewReqRes(ocabci.ToRequestFlush(), cb)
	return app.done(reqRes, ocabci.ToResponseFlush())
}

func (app *localClient) EchoAsync(msg string, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestEcho(msg), cb)
	return app.done(reqRes, ocabci.ToResponseEcho(msg))
}

func (app *localClient) InfoAsync(req abci.RequestInfo, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestInfo(req), cb)
	res := app.Application.Info(req)
	return app.done(reqRes, ocabci.ToResponseInfo(res))
}

func (app *localClient) SetOptionAsync(req abci.RequestSetOption, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestSetOption(req), cb)
	res := app.Application.SetOption(req)
	return app.done(reqRes, ocabci.ToResponseSetOption(res))
}

func (app *localClient) DeliverTxAsync(req abci.RequestDeliverTx, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestDeliverTx(req), cb)
	res := app.Application.DeliverTx(req)
	return app.done(reqRes, ocabci.ToResponseDeliverTx(res))
}

func (app *localClient) CheckTxAsync(req abci.RequestCheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestCheckTx(req), cb)

	app.Application.CheckTxAsync(req, func(r ocabci.ResponseCheckTx) {
		res := ocabci.ToResponseCheckTx(r)
		app.done(reqRes, res)
	})

	return reqRes
}

func (app *localClient) QueryAsync(req abci.RequestQuery, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestQuery(req), cb)
	res := app.Application.Query(req)
	return app.done(reqRes, ocabci.ToResponseQuery(res))
}

func (app *localClient) CommitAsync(cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestCommit(), cb)
	res := app.Application.Commit()
	return app.done(reqRes, ocabci.ToResponseCommit(res))
}

func (app *localClient) InitChainAsync(req ocabci.RequestInitChain, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestInitChain(req), cb)
	res := app.Application.InitChain(req)
	return app.done(reqRes, ocabci.ToResponseInitChain(res))
}

func (app *localClient) BeginBlockAsync(req ocabci.RequestBeginBlock, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestBeginBlock(req), cb)
	res := app.Application.BeginBlock(req)
	return app.done(reqRes, ocabci.ToResponseBeginBlock(res))
}

func (app *localClient) EndBlockAsync(req abci.RequestEndBlock, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestEndBlock(req), cb)
	res := app.Application.EndBlock(req)
	return app.done(reqRes, ocabci.ToResponseEndBlock(res))
}

func (app *localClient) BeginRecheckTxAsync(req ocabci.RequestBeginRecheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestBeginRecheckTx(req), cb)
	res := app.Application.BeginRecheckTx(req)
	return app.done(reqRes, ocabci.ToResponseBeginRecheckTx(res))
}

func (app *localClient) EndRecheckTxAsync(req ocabci.RequestEndRecheckTx, cb ResponseCallback) *ReqRes {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestEndRecheckTx(req), cb)
	res := app.Application.EndRecheckTx(req)
	return app.done(reqRes, ocabci.ToResponseEndRecheckTx(res))
}

func (app *localClient) ListSnapshotsAsync(req abci.RequestListSnapshots, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestListSnapshots(req), cb)
	res := app.Application.ListSnapshots(req)
	return app.done(reqRes, ocabci.ToResponseListSnapshots(res))
}

func (app *localClient) OfferSnapshotAsync(req abci.RequestOfferSnapshot, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestOfferSnapshot(req), cb)
	res := app.Application.OfferSnapshot(req)
	return app.done(reqRes, ocabci.ToResponseOfferSnapshot(res))
}

func (app *localClient) LoadSnapshotChunkAsync(req abci.RequestLoadSnapshotChunk, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestLoadSnapshotChunk(req), cb)
	res := app.Application.LoadSnapshotChunk(req)
	return app.done(reqRes, ocabci.ToResponseLoadSnapshotChunk(res))
}

func (app *localClient) ApplySnapshotChunkAsync(req abci.RequestApplySnapshotChunk, cb ResponseCallback) *ReqRes {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	reqRes := NewReqRes(ocabci.ToRequestApplySnapshotChunk(req), cb)
	res := app.Application.ApplySnapshotChunk(req)
	return app.done(reqRes, ocabci.ToResponseApplySnapshotChunk(res))
}

//-------------------------------------------------------
func (app *localClient) FlushSync() (*abci.ResponseFlush, error) {
	return &abci.ResponseFlush{}, nil
}

func (app *localClient) EchoSync(msg string) (*abci.ResponseEcho, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	return &abci.ResponseEcho{Message: msg}, nil
}

func (app *localClient) InfoSync(req abci.RequestInfo) (*abci.ResponseInfo, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Info(req)
	return &res, nil
}

func (app *localClient) SetOptionSync(req abci.RequestSetOption) (*abci.ResponseSetOption, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.SetOption(req)
	return &res, nil
}

func (app *localClient) DeliverTxSync(req abci.RequestDeliverTx) (*abci.ResponseDeliverTx, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.DeliverTx(req)
	return &res, nil
}

func (app *localClient) CheckTxSync(req abci.RequestCheckTx) (*ocabci.ResponseCheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.CheckTxSync(req)
	return &res, nil
}

func (app *localClient) QuerySync(req abci.RequestQuery) (*abci.ResponseQuery, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Query(req)
	return &res, nil
}

func (app *localClient) CommitSync() (*abci.ResponseCommit, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.Commit()
	return &res, nil
}

func (app *localClient) InitChainSync(req ocabci.RequestInitChain) (*ocabci.ResponseInitChain, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.InitChain(req)
	return &res, nil
}

func (app *localClient) BeginBlockSync(req ocabci.RequestBeginBlock) (*abci.ResponseBeginBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.BeginBlock(req)
	return &res, nil
}

func (app *localClient) EndBlockSync(req abci.RequestEndBlock) (*ocabci.ResponseEndBlock, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.EndBlock(req)
	return &res, nil
}

func (app *localClient) BeginRecheckTxSync(req ocabci.RequestBeginRecheckTx) (*ocabci.ResponseBeginRecheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.BeginRecheckTx(req)
	return &res, nil
}

func (app *localClient) EndRecheckTxSync(req ocabci.RequestEndRecheckTx) (*ocabci.ResponseEndRecheckTx, error) {
	// NOTE: commented out for performance. delete all after commenting out all `app.mtx`
	// app.mtx.Lock()
	// defer app.mtx.Unlock()

	res := app.Application.EndRecheckTx(req)
	return &res, nil
}

func (app *localClient) ListSnapshotsSync(req abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.ListSnapshots(req)
	return &res, nil
}

func (app *localClient) OfferSnapshotSync(req abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.OfferSnapshot(req)
	return &res, nil
}

func (app *localClient) LoadSnapshotChunkSync(
	req abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.LoadSnapshotChunk(req)
	return &res, nil
}

func (app *localClient) ApplySnapshotChunkSync(
	req abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	app.mtx.Lock()
	defer app.mtx.Unlock()

	res := app.Application.ApplySnapshotChunk(req)
	return &res, nil
}

//-------------------------------------------------------

func (app *localClient) done(reqRes *ReqRes, res *ocabci.Response) *ReqRes {
	set := reqRes.SetDone(res)
	if set {
		if globalCb := app.GetGlobalCallback(); globalCb != nil {
			globalCb(reqRes.Request, res)
		}
	}
	return reqRes
}
