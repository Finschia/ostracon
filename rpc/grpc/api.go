package coregrpc

import (
	"context"

	abci "github.com/tendermint/tendermint/abci/types"
	core_grpc "github.com/tendermint/tendermint/rpc/grpc"

	core "github.com/Finschia/ostracon/rpc/core"
	rpctypes "github.com/Finschia/ostracon/rpc/jsonrpc/types"
)

type broadcastAPI struct {
}

func (bapi *broadcastAPI) Ping(ctx context.Context, req *core_grpc.RequestPing) (*core_grpc.ResponsePing, error) {
	// kvstore so we can check if the server is up
	return &core_grpc.ResponsePing{}, nil
}

func (bapi *broadcastAPI) BroadcastTx(ctx context.Context, req *core_grpc.RequestBroadcastTx) (*core_grpc.ResponseBroadcastTx, error) {
	// NOTE: there's no way to get client's remote address
	// see https://stackoverflow.com/questions/33684570/session-and-remote-ip-address-in-grpc-go
	res, err := core.BroadcastTxCommit(&rpctypes.Context{}, req.Tx)
	if err != nil {
		return nil, err
	}

	return &core_grpc.ResponseBroadcastTx{
		CheckTx: &abci.ResponseCheckTx{
			Code: res.CheckTx.Code,
			Data: res.CheckTx.Data,
			Log:  res.CheckTx.Log,
		},
		DeliverTx: &abci.ResponseDeliverTx{
			Code: res.DeliverTx.Code,
			Data: res.DeliverTx.Data,
			Log:  res.DeliverTx.Log,
		},
	}, nil
}
