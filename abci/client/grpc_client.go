package abcicli

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	tmabci "github.com/tendermint/tendermint/abci/types"

	abci "github.com/line/ostracon/abci/types"
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

	client abci.ABCIApplicationClient
	conn   *grpc.ClientConn

	mtx  tmsync.Mutex
	addr string
	err  error

	globalCbMtx sync.Mutex
	globalCb    func(*abci.Request, *abci.Response) // listens to all callbacks
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
		client := abci.NewABCIApplicationClient(conn)
		cli.conn = conn

	ENSURE_CONNECTED:
		for {
			_, err := client.Echo(context.Background(), &tmabci.RequestEcho{Message: "hello"}, grpc.WaitForReady(true))
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
	req := abci.ToRequestEcho(msg)
	res, err := cli.client.Echo(context.Background(), req.GetEcho(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_Echo{Echo: res}}, cb)
}

func (cli *grpcClient) FlushAsync(cb ResponseCallback) *ReqRes {
	req := abci.ToRequestFlush()
	res, err := cli.client.Flush(context.Background(), req.GetFlush(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_Flush{Flush: res}}, cb)
}

func (cli *grpcClient) InfoAsync(params tmabci.RequestInfo, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestInfo(params)
	res, err := cli.client.Info(context.Background(), req.GetInfo(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_Info{Info: res}}, cb)
}

func (cli *grpcClient) SetOptionAsync(params tmabci.RequestSetOption, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestSetOption(params)
	res, err := cli.client.SetOption(context.Background(), req.GetSetOption(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_SetOption{SetOption: res}}, cb)
}

func (cli *grpcClient) DeliverTxAsync(params tmabci.RequestDeliverTx, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestDeliverTx(params)
	res, err := cli.client.DeliverTx(context.Background(), req.GetDeliverTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_DeliverTx{DeliverTx: res}}, cb)
}

func (cli *grpcClient) CheckTxAsync(params tmabci.RequestCheckTx, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestCheckTx(params)
	res, err := cli.client.CheckTx(context.Background(), req.GetCheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_CheckTx{CheckTx: res}}, cb)
}

func (cli *grpcClient) QueryAsync(params tmabci.RequestQuery, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestQuery(params)
	res, err := cli.client.Query(context.Background(), req.GetQuery(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_Query{Query: res}}, cb)
}

func (cli *grpcClient) CommitAsync(cb ResponseCallback) *ReqRes {
	req := abci.ToRequestCommit()
	res, err := cli.client.Commit(context.Background(), req.GetCommit(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_Commit{Commit: res}}, cb)
}

func (cli *grpcClient) InitChainAsync(params abci.RequestInitChain, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestInitChain(params)
	res, err := cli.client.InitChain(context.Background(), req.GetInitChain(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_InitChain{InitChain: res}}, cb)
}

func (cli *grpcClient) BeginBlockAsync(params abci.RequestBeginBlock, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestBeginBlock(params)
	res, err := cli.client.BeginBlock(context.Background(), req.GetBeginBlock(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_BeginBlock{BeginBlock: res}}, cb)
}

func (cli *grpcClient) EndBlockAsync(params tmabci.RequestEndBlock, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestEndBlock(params)
	res, err := cli.client.EndBlock(context.Background(), req.GetEndBlock(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_EndBlock{EndBlock: res}}, cb)
}

func (cli *grpcClient) BeginRecheckTxAsync(params abci.RequestBeginRecheckTx, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestBeginRecheckTx(params)
	res, err := cli.client.BeginRecheckTx(context.Background(), req.GetBeginRecheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_BeginRecheckTx{BeginRecheckTx: res}}, cb)
}

func (cli *grpcClient) EndRecheckTxAsync(params abci.RequestEndRecheckTx, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestEndRecheckTx(params)
	res, err := cli.client.EndRecheckTx(context.Background(), req.GetEndRecheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_EndRecheckTx{EndRecheckTx: res}}, cb)
}

func (cli *grpcClient) ListSnapshotsAsync(params tmabci.RequestListSnapshots, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestListSnapshots(params)
	res, err := cli.client.ListSnapshots(context.Background(), req.GetListSnapshots(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_ListSnapshots{ListSnapshots: res}}, cb)
}

func (cli *grpcClient) OfferSnapshotAsync(params tmabci.RequestOfferSnapshot, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestOfferSnapshot(params)
	res, err := cli.client.OfferSnapshot(context.Background(), req.GetOfferSnapshot(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_OfferSnapshot{OfferSnapshot: res}}, cb)
}

func (cli *grpcClient) LoadSnapshotChunkAsync(params tmabci.RequestLoadSnapshotChunk, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestLoadSnapshotChunk(params)
	res, err := cli.client.LoadSnapshotChunk(context.Background(), req.GetLoadSnapshotChunk(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &abci.Response{Value: &abci.Response_LoadSnapshotChunk{LoadSnapshotChunk: res}}, cb)
}

func (cli *grpcClient) ApplySnapshotChunkAsync(params tmabci.RequestApplySnapshotChunk, cb ResponseCallback) *ReqRes {
	req := abci.ToRequestApplySnapshotChunk(params)
	res, err := cli.client.ApplySnapshotChunk(context.Background(), req.GetApplySnapshotChunk(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req,
		&abci.Response{Value: &abci.Response_ApplySnapshotChunk{ApplySnapshotChunk: res}}, cb)
}

func (cli *grpcClient) finishAsyncCall(req *abci.Request, res *abci.Response, cb ResponseCallback) *ReqRes {
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

//----------------------------------------
func (cli *grpcClient) FlushSync() (*tmabci.ResponseFlush, error) {
	reqres := cli.FlushAsync(nil)
	reqres.Wait()
	return reqres.Response.GetFlush(), cli.Error()
}

func (cli *grpcClient) EchoSync(msg string) (*tmabci.ResponseEcho, error) {
	reqres := cli.EchoAsync(msg, nil)
	reqres.Wait()
	// StopForError should already have been called if error is set
	return reqres.Response.GetEcho(), cli.Error()
}

func (cli *grpcClient) InfoSync(req tmabci.RequestInfo) (*tmabci.ResponseInfo, error) {
	reqres := cli.InfoAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetInfo(), cli.Error()
}

func (cli *grpcClient) SetOptionSync(req tmabci.RequestSetOption) (*tmabci.ResponseSetOption, error) {
	reqres := cli.SetOptionAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetSetOption(), cli.Error()
}

func (cli *grpcClient) DeliverTxSync(params tmabci.RequestDeliverTx) (*tmabci.ResponseDeliverTx, error) {
	reqres := cli.DeliverTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetDeliverTx(), cli.Error()
}

func (cli *grpcClient) CheckTxSync(params tmabci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	reqres := cli.CheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetCheckTx(), cli.Error()
}

func (cli *grpcClient) QuerySync(req tmabci.RequestQuery) (*tmabci.ResponseQuery, error) {
	reqres := cli.QueryAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetQuery(), cli.Error()
}

func (cli *grpcClient) CommitSync() (*tmabci.ResponseCommit, error) {
	reqres := cli.CommitAsync(nil)
	reqres.Wait()
	return reqres.Response.GetCommit(), cli.Error()
}

func (cli *grpcClient) InitChainSync(params abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	reqres := cli.InitChainAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetInitChain(), cli.Error()
}

func (cli *grpcClient) BeginBlockSync(params abci.RequestBeginBlock) (*tmabci.ResponseBeginBlock, error) {
	reqres := cli.BeginBlockAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetBeginBlock(), cli.Error()
}

func (cli *grpcClient) EndBlockSync(params tmabci.RequestEndBlock) (*abci.ResponseEndBlock, error) {
	reqres := cli.EndBlockAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetEndBlock(), cli.Error()
}

func (cli *grpcClient) BeginRecheckTxSync(params abci.RequestBeginRecheckTx) (*abci.ResponseBeginRecheckTx, error) {
	reqres := cli.BeginRecheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetBeginRecheckTx(), cli.Error()
}

func (cli *grpcClient) EndRecheckTxSync(params abci.RequestEndRecheckTx) (*abci.ResponseEndRecheckTx, error) {
	reqres := cli.EndRecheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetEndRecheckTx(), cli.Error()
}

func (cli *grpcClient) ListSnapshotsSync(params tmabci.RequestListSnapshots) (*tmabci.ResponseListSnapshots, error) {
	reqres := cli.ListSnapshotsAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetListSnapshots(), cli.Error()
}

func (cli *grpcClient) OfferSnapshotSync(params tmabci.RequestOfferSnapshot) (*tmabci.ResponseOfferSnapshot, error) {
	reqres := cli.OfferSnapshotAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetOfferSnapshot(), cli.Error()
}

func (cli *grpcClient) LoadSnapshotChunkSync(
	params tmabci.RequestLoadSnapshotChunk) (*tmabci.ResponseLoadSnapshotChunk, error) {
	reqres := cli.LoadSnapshotChunkAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetLoadSnapshotChunk(), cli.Error()
}

func (cli *grpcClient) ApplySnapshotChunkSync(
	params tmabci.RequestApplySnapshotChunk) (*tmabci.ResponseApplySnapshotChunk, error) {
	reqres := cli.ApplySnapshotChunkAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetApplySnapshotChunk(), cli.Error()
}
