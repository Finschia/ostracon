package abcicli

import (
	"fmt"
	"testing"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/line/ostracon/abci/server"
	ocabci "github.com/line/ostracon/abci/types"
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
	c.SetGlobalCallback(func(*ocabci.Request, *ocabci.Response) {
	})
	err0 = c.Start()
	require.NoError(t, err0)

	c.EchoAsync("msg", getResponseCallback(t))
	c.FlushAsync(getResponseCallback(t))
	c.InfoAsync(abci.RequestInfo{}, getResponseCallback(t))
	c.SetOptionAsync(abci.RequestSetOption{}, getResponseCallback(t))
	c.DeliverTxAsync(abci.RequestDeliverTx{}, getResponseCallback(t))
	c.CheckTxAsync(abci.RequestCheckTx{}, getResponseCallback(t))
	c.QueryAsync(abci.RequestQuery{}, getResponseCallback(t))
	c.CommitAsync(getResponseCallback(t))
	c.InitChainAsync(ocabci.RequestInitChain{}, getResponseCallback(t))
	c.BeginBlockAsync(ocabci.RequestBeginBlock{}, getResponseCallback(t))
	c.EndBlockAsync(abci.RequestEndBlock{}, getResponseCallback(t))
	c.BeginRecheckTxAsync(ocabci.RequestBeginRecheckTx{}, getResponseCallback(t))
	c.EndRecheckTxAsync(ocabci.RequestEndRecheckTx{}, getResponseCallback(t))
	c.ListSnapshotsAsync(abci.RequestListSnapshots{}, getResponseCallback(t))
	c.OfferSnapshotAsync(abci.RequestOfferSnapshot{}, getResponseCallback(t))
	c.LoadSnapshotChunkAsync(abci.RequestLoadSnapshotChunk{}, getResponseCallback(t))
	c.ApplySnapshotChunkAsync(abci.RequestApplySnapshotChunk{}, getResponseCallback(t))

	_, err := c.EchoSync("msg")
	require.NoError(t, err)

	_, err = c.FlushSync()
	require.NoError(t, err)

	_, err = c.InfoSync(abci.RequestInfo{})
	require.NoError(t, err)

	_, err = c.SetOptionSync(abci.RequestSetOption{})
	require.NoError(t, err)

	_, err = c.DeliverTxSync(abci.RequestDeliverTx{})
	require.NoError(t, err)

	_, err = c.CheckTxSync(abci.RequestCheckTx{})
	require.NoError(t, err)

	_, err = c.QuerySync(abci.RequestQuery{})
	require.NoError(t, err)

	_, err = c.CommitSync()
	require.NoError(t, err)

	_, err = c.InitChainSync(ocabci.RequestInitChain{})
	require.NoError(t, err)

	_, err = c.BeginBlockSync(ocabci.RequestBeginBlock{})
	require.NoError(t, err)

	_, err = c.EndBlockSync(abci.RequestEndBlock{})
	require.NoError(t, err)

	_, err = c.BeginRecheckTxSync(ocabci.RequestBeginRecheckTx{})
	require.NoError(t, err)

	_, err = c.EndRecheckTxSync(ocabci.RequestEndRecheckTx{})
	require.NoError(t, err)

	_, err = c.ListSnapshotsSync(abci.RequestListSnapshots{})
	require.NoError(t, err)

	_, err = c.OfferSnapshotSync(abci.RequestOfferSnapshot{})
	require.NoError(t, err)

	_, err = c.LoadSnapshotChunkSync(abci.RequestLoadSnapshotChunk{})
	require.NoError(t, err)

	_, err = c.ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk{})
	require.NoError(t, err)
}
