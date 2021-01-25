package abcicli

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/tendermint/tendermint/abci/types"
	tmnet "github.com/tendermint/tendermint/libs/net"
	"github.com/tendermint/tendermint/libs/service"
)

var _ Client = (*grpcClient)(nil)

// A stripped copy of the remoteClient that makes
// synchronous calls using grpc
type grpcClient struct {
	service.BaseService
	mustConnect bool

	client types.ABCIApplicationClient
	conn   *grpc.ClientConn

	mtx  sync.Mutex
	addr string
	err  error

	globalCbMtx sync.Mutex
	globalCb    func(*types.Request, *types.Response) // listens to all callbacks
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
		client := types.NewABCIApplicationClient(conn)
		cli.conn = conn

	ENSURE_CONNECTED:
		for {
			_, err := client.Echo(context.Background(), &types.RequestEcho{Message: "hello"}, grpc.WaitForReady(true))
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
	cli.Stop()
}

func (cli *grpcClient) Error() error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	return cli.err
}

func (cli *grpcClient) SetGlobalCallback(globalCb GlobalCallback) {
	cli.globalCbMtx.Lock()
	cli.globalCb = globalCb
	cli.globalCbMtx.Unlock()
}

func (cli *grpcClient) GetGlobalCallback() (cb GlobalCallback) {
	cli.globalCbMtx.Lock()
	cb = cli.globalCb
	cli.globalCbMtx.Unlock()
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
	req := types.ToRequestEcho(msg)
	res, err := cli.client.Echo(context.Background(), req.GetEcho(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_Echo{Echo: res}}, cb)
}

func (cli *grpcClient) InfoAsync(params types.RequestInfo, cb ResponseCallback) *ReqRes {
	req := types.ToRequestInfo(params)
	res, err := cli.client.Info(context.Background(), req.GetInfo(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_Info{Info: res}}, cb)
}

func (cli *grpcClient) SetOptionAsync(params types.RequestSetOption, cb ResponseCallback) *ReqRes {
	req := types.ToRequestSetOption(params)
	res, err := cli.client.SetOption(context.Background(), req.GetSetOption(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_SetOption{SetOption: res}}, cb)
}

func (cli *grpcClient) DeliverTxAsync(params types.RequestDeliverTx, cb ResponseCallback) *ReqRes {
	req := types.ToRequestDeliverTx(params)
	res, err := cli.client.DeliverTx(context.Background(), req.GetDeliverTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_DeliverTx{DeliverTx: res}}, cb)
}

func (cli *grpcClient) CheckTxAsync(params types.RequestCheckTx, cb ResponseCallback) *ReqRes {
	req := types.ToRequestCheckTx(params)
	res, err := cli.client.CheckTx(context.Background(), req.GetCheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_CheckTx{CheckTx: res}}, cb)
}

func (cli *grpcClient) QueryAsync(params types.RequestQuery, cb ResponseCallback) *ReqRes {
	req := types.ToRequestQuery(params)
	res, err := cli.client.Query(context.Background(), req.GetQuery(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_Query{Query: res}}, cb)
}

func (cli *grpcClient) CommitAsync(cb ResponseCallback) *ReqRes {
	req := types.ToRequestCommit()
	res, err := cli.client.Commit(context.Background(), req.GetCommit(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_Commit{Commit: res}}, cb)
}

func (cli *grpcClient) InitChainAsync(params types.RequestInitChain, cb ResponseCallback) *ReqRes {
	req := types.ToRequestInitChain(params)
	res, err := cli.client.InitChain(context.Background(), req.GetInitChain(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_InitChain{InitChain: res}}, cb)
}

func (cli *grpcClient) BeginBlockAsync(params types.RequestBeginBlock, cb ResponseCallback) *ReqRes {
	req := types.ToRequestBeginBlock(params)
	res, err := cli.client.BeginBlock(context.Background(), req.GetBeginBlock(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_BeginBlock{BeginBlock: res}}, cb)
}

func (cli *grpcClient) EndBlockAsync(params types.RequestEndBlock, cb ResponseCallback) *ReqRes {
	req := types.ToRequestEndBlock(params)
	res, err := cli.client.EndBlock(context.Background(), req.GetEndBlock(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_EndBlock{EndBlock: res}}, cb)
}

func (cli *grpcClient) BeginRecheckTxAsync(params types.RequestBeginRecheckTx, cb ResponseCallback) *ReqRes {
	req := types.ToRequestBeginRecheckTx(params)
	res, err := cli.client.BeginRecheckTx(context.Background(), req.GetBeginRecheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_BeginRecheckTx{BeginRecheckTx: res}}, cb)
}

func (cli *grpcClient) EndRecheckTxAsync(params types.RequestEndRecheckTx, cb ResponseCallback) *ReqRes {
	req := types.ToRequestEndRecheckTx(params)
	res, err := cli.client.EndRecheckTx(context.Background(), req.GetEndRecheckTx(), grpc.WaitForReady(true))
	if err != nil {
		cli.StopForError(err)
	}
	return cli.finishAsyncCall(req, &types.Response{Value: &types.Response_EndRecheckTx{EndRecheckTx: res}}, cb)
}

func (cli *grpcClient) finishAsyncCall(req *types.Request, res *types.Response, cb ResponseCallback) *ReqRes {
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

func (cli *grpcClient) FlushSync() error {
	return nil
}

func (cli *grpcClient) EchoSync(msg string) (*types.ResponseEcho, error) {
	reqres := cli.EchoAsync(msg, nil)
	reqres.Wait()
	// StopForError should already have been called if error is set
	return reqres.Response.GetEcho(), cli.Error()
}

func (cli *grpcClient) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	reqres := cli.InfoAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetInfo(), cli.Error()
}

func (cli *grpcClient) SetOptionSync(req types.RequestSetOption) (*types.ResponseSetOption, error) {
	reqres := cli.SetOptionAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetSetOption(), cli.Error()
}

func (cli *grpcClient) DeliverTxSync(params types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	reqres := cli.DeliverTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetDeliverTx(), cli.Error()
}

func (cli *grpcClient) CheckTxSync(params types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	reqres := cli.CheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetCheckTx(), cli.Error()
}

func (cli *grpcClient) QuerySync(req types.RequestQuery) (*types.ResponseQuery, error) {
	reqres := cli.QueryAsync(req, nil)
	reqres.Wait()
	return reqres.Response.GetQuery(), cli.Error()
}

func (cli *grpcClient) CommitSync() (*types.ResponseCommit, error) {
	reqres := cli.CommitAsync(nil)
	reqres.Wait()
	return reqres.Response.GetCommit(), cli.Error()
}

func (cli *grpcClient) InitChainSync(params types.RequestInitChain) (*types.ResponseInitChain, error) {
	reqres := cli.InitChainAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetInitChain(), cli.Error()
}

func (cli *grpcClient) BeginBlockSync(params types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	reqres := cli.BeginBlockAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetBeginBlock(), cli.Error()
}

func (cli *grpcClient) EndBlockSync(params types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	reqres := cli.EndBlockAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetEndBlock(), cli.Error()
}

func (cli *grpcClient) BeginRecheckTxSync(params types.RequestBeginRecheckTx) (*types.ResponseBeginRecheckTx, error) {
	reqres := cli.BeginRecheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetBeginRecheckTx(), cli.Error()
}

func (cli *grpcClient) EndRecheckTxSync(params types.RequestEndRecheckTx) (*types.ResponseEndRecheckTx, error) {
	reqres := cli.EndRecheckTxAsync(params, nil)
	reqres.Wait()
	return reqres.Response.GetEndRecheckTx(), cli.Error()
}
