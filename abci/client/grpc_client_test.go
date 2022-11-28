package abcicli

import (
	"fmt"
	"testing"

	"github.com/line/ostracon/abci/server"
	"github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/rand"
	"github.com/stretchr/testify/require"
)

func TestGrpcClientCalls(t *testing.T) {
	app := sampleApp{}

	port := 20000 + rand.Int32()%10000
	addr := fmt.Sprintf("localhost:%d", port)

	s, err0 := server.NewServer(addr, "grpc", app)
	require.NoError(t, err0)
	err0 = s.Start()
	require.NoError(t, err0)

	c := NewGRPCClient(addr, true)
	c.SetResponseCallback(func(*types.Request, *types.Response) {
	})
	err0 = c.Start()
	require.NoError(t, err0)

	c.EchoAsync("msg")
	c.FlushAsync()
	c.InfoAsync(types.RequestInfo{})
	c.SetOptionAsync(types.RequestSetOption{})
	c.DeliverTxAsync(types.RequestDeliverTx{})
	c.CheckTxAsync(types.RequestCheckTx{})
	c.QueryAsync(types.RequestQuery{})
	c.CommitAsync()
	c.InitChainAsync(types.RequestInitChain{})
	c.BeginBlockAsync(types.RequestBeginBlock{})
	c.EndBlockAsync(types.RequestEndBlock{})
	c.ListSnapshotsAsync(types.RequestListSnapshots{})
	c.OfferSnapshotAsync(types.RequestOfferSnapshot{})
	c.LoadSnapshotChunkAsync(types.RequestLoadSnapshotChunk{})
	c.ApplySnapshotChunkAsync(types.RequestApplySnapshotChunk{})

	_, err := c.EchoSync("msg")
	require.NoError(t, err)

	err = c.FlushSync()
	require.NoError(t, err)

	_, err = c.InfoSync(types.RequestInfo{})
	require.NoError(t, err)

	_, err = c.SetOptionSync(types.RequestSetOption{})
	require.NoError(t, err)

	_, err = c.DeliverTxSync(types.RequestDeliverTx{})
	require.NoError(t, err)

	_, err = c.CheckTxSync(types.RequestCheckTx{})
	require.NoError(t, err)

	_, err = c.QuerySync(types.RequestQuery{})
	require.NoError(t, err)

	_, err = c.CommitSync()
	require.NoError(t, err)

	_, err = c.InitChainSync(types.RequestInitChain{})
	require.NoError(t, err)

	_, err = c.BeginBlockSync(types.RequestBeginBlock{})
	require.NoError(t, err)

	_, err = c.EndBlockSync(types.RequestEndBlock{})
	require.NoError(t, err)

	_, err = c.ListSnapshotsSync(types.RequestListSnapshots{})
	require.NoError(t, err)

	_, err = c.OfferSnapshotSync(types.RequestOfferSnapshot{})
	require.NoError(t, err)

	_, err = c.LoadSnapshotChunkSync(types.RequestLoadSnapshotChunk{})
	require.NoError(t, err)

	_, err = c.ApplySnapshotChunkSync(types.RequestApplySnapshotChunk{})
	require.NoError(t, err)
}
