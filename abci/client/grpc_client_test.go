package abcicli

import (
	"fmt"
	"testing"

	"github.com/tendermint/tendermint/abci/types"

	"github.com/Finschia/ostracon/abci/server"
	ocabci "github.com/Finschia/ostracon/abci/types"
	"github.com/Finschia/ostracon/libs/rand"
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
	c.SetGlobalCallback(func(*ocabci.Request, *ocabci.Response) {
	})
	err0 = c.Start()
	require.NoError(t, err0)

	c.EchoAsync("msg", getResponseCallback(t))
	c.FlushAsync(getResponseCallback(t))
	c.InfoAsync(types.RequestInfo{}, getResponseCallback(t))
	c.SetOptionAsync(types.RequestSetOption{}, getResponseCallback(t))
	c.DeliverTxAsync(types.RequestDeliverTx{}, getResponseCallback(t))
	c.CheckTxAsync(types.RequestCheckTx{}, getResponseCallback(t))
	c.QueryAsync(types.RequestQuery{}, getResponseCallback(t))
	c.CommitAsync(getResponseCallback(t))
	c.InitChainAsync(types.RequestInitChain{}, getResponseCallback(t))
	c.BeginBlockAsync(ocabci.RequestBeginBlock{}, getResponseCallback(t))
	c.EndBlockAsync(types.RequestEndBlock{}, getResponseCallback(t))
	c.BeginRecheckTxAsync(ocabci.RequestBeginRecheckTx{}, getResponseCallback(t))
	c.EndRecheckTxAsync(ocabci.RequestEndRecheckTx{}, getResponseCallback(t))
	c.ListSnapshotsAsync(types.RequestListSnapshots{}, getResponseCallback(t))
	c.OfferSnapshotAsync(types.RequestOfferSnapshot{}, getResponseCallback(t))
	c.LoadSnapshotChunkAsync(types.RequestLoadSnapshotChunk{}, getResponseCallback(t))
	c.ApplySnapshotChunkAsync(types.RequestApplySnapshotChunk{}, getResponseCallback(t))

	_, err := c.EchoSync("msg")
	require.NoError(t, err)

	_, err = c.FlushSync()
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

	_, err = c.BeginBlockSync(ocabci.RequestBeginBlock{})
	require.NoError(t, err)

	_, err = c.EndBlockSync(types.RequestEndBlock{})
	require.NoError(t, err)

	_, err = c.BeginRecheckTxSync(ocabci.RequestBeginRecheckTx{})
	require.NoError(t, err)

	_, err = c.EndRecheckTxSync(ocabci.RequestEndRecheckTx{})
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
