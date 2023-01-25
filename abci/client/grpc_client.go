package abcicli

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	abci "github.com/tendermint/tendermint/abci/types"

	ocabci "github.com/line/ostracon/abci/types"
	tmnet "github.com/line/ostracon/libs/net"
	"github.com/line/ostracon/libs/service"
	tmsync "github.com/line/ostracon/libs/sync"
)

var _ Client = (*grpcClient)(nil)

// A stripped copy of the remoteClient that makes
// synchronous calls using grpc
type grpcClient struct {
	service.BaseService
	mustConnect bool

	client ocabci.ABCIApplicationClient
	conn   *grpc.ClientConn

	mtx  tmsync.Mutex
	addr string
	err  error

	globalCbMtx sync.Mutex
	globalCb    func(*ocabci.Request, *ocabci.Response) // listens to all callbacks
}

func NewGRPCClient(addr string, mustConnect bool) Client {
	cli := &grpcClient{
		addr:        addr,
		mustConnect: mustConnect,
	}
	cli.BaseService = *service.NewBaseService(nil, "grpcClient", cli)
	return cli
}

func dialerFunc(ctx context.Context, addr string) (net.Conn, error) {
	return tmnet.Connect(addr)
}

func (cli *grpcClient) OnStart() error {
	if err := cli.BaseService.OnStart(); err != nil {
		return err
	}

RETRY_LOOP:
	for {
		//nolint:staticcheck // SA1019 Existing use of deprecated but supported dial option.
		conn, err := grpc.Dial(cli.addr, grpc.WithInsecure(), grpc.WithContextDialer(dialerFunc))
		if err != nil {
			if cli.mustConnect {
				return err
			}
			cli.Logger.Error(fmt.Sprintf("abci.grpcClient failed to connect to %v.  Retrying...\n", cli.addr), "err", err)
			time.Sleep(time.Second * dialRetryIntervalSeconds)
			continue RETRY_LOOP
		}

		cli.Logger.Info("Dialed server. Waiting for echo.", "addr", cli.addr)
		client := ocabci.NewABCIApplicationClient(conn)
		cli.conn = conn

	ENSURE_CONNECTED:
		for {
			_, err := client.Echo(context.Background(), &abci.RequestEcho{Message: "hello"}, grpc.WaitForReady(true))
			if err == nil {
				break ENSURE_CONNECTED
			}
			cli.Logger.Error("Echo failed", "err", err)
			time.Sleep(time.Second * echoRetryIntervalSeconds)
		}

		cli.client = client
		return nil
	}
}

func (cli *grpcClient) OnStop() {
	cli.BaseService.OnStop()

	if cli.conn != nil {
		cli.conn.Close()
	}
}

func (cli *grpcClient) StopForError(err error) {
	if !cli.IsRunning() {
		return
	}

	cli.mtx.Lock()
	if cli.err == nil {
		cli.err = err
	}
	cli.mtx.Unlock()

	cli.Logger.Error(fmt.Sprintf("Stopping abci.grpcClient for error: %v", err.Error()))
	if err := cli.Stop(); err != nil {
		cli.Logger.Error("Error stopping abci.grpcClient", "err", err)
	}
}

func (cli *grpcClient) Error() error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	return cli.err
}

func (cli *grpcClient) SetGlobalCallback(globalCb GlobalCallback) {
	cli.globalCbMtx.Lock()
	defer cli.globalCbMtx.Unlock()
	cli.globalCb = globalCb
}

func (cli *grpcClient) GetGlobalCallback() (cb GlobalCallback) {
	cli.globalCbMtx.Lock()
	defer cli.globalCbMtx.Unlock()
	cb = cli.globalCb
	return cb
}

//----------------------------------------
// GRPC calls are synchronous, but some callbacks expect to be called asynchronously
// (eg. the mempool expects to be able to lock to remove bad txs from cache).
// To accommodate, we finish each call in its own go-routine,
// which is expensive, but easy - if you want something better, use the socket protocol!
// maybe one day, if people really want it, we use grpc streams,
// but hopefully not :D

func (cli *grpcClient) EchoAsync(msg string, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestEcho(msg)
	res, err := cli.client.Echo(context.Background(), req.GetEcho(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_Echo{Echo: res}}, cb)
}

func (cli *grpcClient) FlushAsync(cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestFlush()
	res, err := cli.client.Flush(context.Background(), req.GetFlush(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_Flush{Flush: res}}, cb)
}

func (cli *grpcClient) InfoAsync(params abci.RequestInfo, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestInfo(params)
	res, err := cli.client.Info(context.Background(), req.GetInfo(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_Info{Info: res}}, cb)
}

func (cli *grpcClient) SetOptionAsync(params abci.RequestSetOption, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestSetOption(params)
	res, err := cli.client.SetOption(context.Background(), req.GetSetOption(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_SetOption{SetOption: res}}, cb)
}

func (cli *grpcClient) DeliverTxAsync(params abci.RequestDeliverTx, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestDeliverTx(params)
	res, err := cli.client.DeliverTx(context.Background(), req.GetDeliverTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_DeliverTx{DeliverTx: res}}, cb)
}

func (cli *grpcClient) CheckTxAsync(params abci.RequestCheckTx, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestCheckTx(params)
	res, err := cli.client.CheckTx(context.Background(), req.GetCheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_CheckTx{CheckTx: res}}, cb)
}

func (cli *grpcClient) QueryAsync(params abci.RequestQuery, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestQuery(params)
	res, err := cli.client.Query(context.Background(), req.GetQuery(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_Query{Query: res}}, cb)
}

func (cli *grpcClient) CommitAsync(cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestCommit()
	res, err := cli.client.Commit(context.Background(), req.GetCommit(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_Commit{Commit: res}}, cb)
}

func (cli *grpcClient) InitChainAsync(params abci.RequestInitChain, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestInitChain(params)
	res, err := cli.client.InitChain(context.Background(), req.GetInitChain(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_InitChain{InitChain: res}}, cb)
}

func (cli *grpcClient) BeginBlockAsync(params ocabci.RequestBeginBlock, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestBeginBlock(params)
	res, err := cli.client.BeginBlock(context.Background(), req.GetBeginBlock(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_BeginBlock{BeginBlock: res}}, cb)
}

func (cli *grpcClient) EndBlockAsync(params abci.RequestEndBlock, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestEndBlock(params)
	res, err := cli.client.EndBlock(context.Background(), req.GetEndBlock(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_EndBlock{EndBlock: res}}, cb)
}

func (cli *grpcClient) BeginRecheckTxAsync(params ocabci.RequestBeginRecheckTx, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestBeginRecheckTx(params)
	res, err := cli.client.BeginRecheckTx(context.Background(), req.GetBeginRecheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_BeginRecheckTx{BeginRecheckTx: res}}, cb)
}

func (cli *grpcClient) EndRecheckTxAsync(params ocabci.RequestEndRecheckTx, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestEndRecheckTx(params)
	res, err := cli.client.EndRecheckTx(context.Background(), req.GetEndRecheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_EndRecheckTx{EndRecheckTx: res}}, cb)
}

func (cli *grpcClient) ListSnapshotsAsync(params abci.RequestListSnapshots, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestListSnapshots(params)
	res, err := cli.client.ListSnapshots(context.Background(), req.GetListSnapshots(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_ListSnapshots{ListSnapshots: res}}, cb)
}

func (cli *grpcClient) OfferSnapshotAsync(params abci.RequestOfferSnapshot, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestOfferSnapshot(params)
	res, err := cli.client.OfferSnapshot(context.Background(), req.GetOfferSnapshot(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_OfferSnapshot{OfferSnapshot: res}}, cb)
}

func (cli *grpcClient) LoadSnapshotChunkAsync(params abci.RequestLoadSnapshotChunk, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestLoadSnapshotChunk(params)
	res, err := cli.client.LoadSnapshotChunk(context.Background(), req.GetLoadSnapshotChunk(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &ocabci.Response{Value: &ocabci.Response_LoadSnapshotChunk{LoadSnapshotChunk: res}}, cb)
}

func (cli *grpcClient) ApplySnapshotChunkAsync(params abci.RequestApplySnapshotChunk, cb ResponseCallback) *ReqRes {
	req := ocabci.ToRequestApplySnapshotChunk(params)
	res, err := cli.client.ApplySnapshotChunk(context.Background(), req.GetApplySnapshotChunk(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req,
		&ocabci.Response{Value: &ocabci.Response_ApplySnapshotChunk{ApplySnapshotChunk: res}}, cb)
}

func (cli *grpcClient) finishAsyncCall(req *ocabci.Request, res *ocabci.Response, cb ResponseCallback) *ReqRes {
	reqRes := NewReqRes(req, cb)

	// goroutine for callbacks
	go func() {
		set := reqRes.SetDone(res)
		if set {
			// Notify client listener if set
			if globalCb := cli.GetGlobalCallback(); globalCb != nil {
				globalCb(req, res)
			}
		}
	}()

	return reqRes
}

// ----------------------------------------
func (cli *grpcClient) FlushSync() (*abci.ResponseFlush, error) {
	reqres := cli.FlushAsync(nil)
	reqres.Wait()
	return reqres.Response.GetFlush(), cli.Error()
}

func (cli *grpcClient) EchoSync(msg string) (*abci.ResponseEcho, error) {
	reqres := cli.EchoAsync(msg, nil)
	reqres.Wait()
	// StopForError should already have been called if error is set
	return reqres.Response.GetEcho(), cli.Error()
}

func (cli *grpcClient) InfoSync(req abci.RequestInfo) (*abci.ResponseInfo, error) {
	reqres := cli.InfoAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetInfo(), cli.Error()
}

func (cli *grpcClient) SetOptionSync(req abci.RequestSetOption) (*abci.ResponseSetOption, error) {
	reqres := cli.SetOptionAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetSetOption(), cli.Error()
}

func (cli *grpcClient) DeliverTxSync(params abci.RequestDeliverTx) (*abci.ResponseDeliverTx, error) {
	reqres := cli.DeliverTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetDeliverTx(), cli.Error()
}

func (cli *grpcClient) CheckTxSync(params abci.RequestCheckTx) (*ocabci.ResponseCheckTx, error) {
	reqres := cli.CheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetCheckTx(), cli.Error()
}

func (cli *grpcClient) QuerySync(req abci.RequestQuery) (*abci.ResponseQuery, error) {
	reqres := cli.QueryAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetQuery(), cli.Error()
}

func (cli *grpcClient) CommitSync() (*abci.ResponseCommit, error) {
	reqres := cli.CommitAsync(nil)
	reqres.Wait()
	return reqres.Response.GetCommit(), cli.Error()
}

func (cli *grpcClient) InitChainSync(params abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	reqres := cli.InitChainAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetInitChain(), cli.Error()
}

func (cli *grpcClient) BeginBlockSync(params ocabci.RequestBeginBlock) (*abci.ResponseBeginBlock, error) {
	reqres := cli.BeginBlockAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetBeginBlock(), cli.Error()
}

func (cli *grpcClient) EndBlockSync(params abci.RequestEndBlock) (*abci.ResponseEndBlock, error) {
	reqres := cli.EndBlockAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetEndBlock(), cli.Error()
}

func (cli *grpcClient) BeginRecheckTxSync(params ocabci.RequestBeginRecheckTx) (*ocabci.ResponseBeginRecheckTx, error) {
	reqres := cli.BeginRecheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetBeginRecheckTx(), cli.Error()
}

func (cli *grpcClient) EndRecheckTxSync(params ocabci.RequestEndRecheckTx) (*ocabci.ResponseEndRecheckTx, error) {
	reqres := cli.EndRecheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetEndRecheckTx(), cli.Error()
}

func (cli *grpcClient) ListSnapshotsSync(params abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	reqres := cli.ListSnapshotsAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetListSnapshots(), cli.Error()
}

func (cli *grpcClient) OfferSnapshotSync(params abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	reqres := cli.OfferSnapshotAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetOfferSnapshot(), cli.Error()
}

func (cli *grpcClient) LoadSnapshotChunkSync(
	params abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	reqres := cli.LoadSnapshotChunkAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetLoadSnapshotChunk(), cli.Error()
}

func (cli *grpcClient) ApplySnapshotChunkSync(
	params abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	reqres := cli.ApplySnapshotChunkAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetApplySnapshotChunk(), cli.Error()
}
