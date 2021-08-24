package abcicli

import (
	"testing"

	"github.com/line/ostracon/abci/types"
	"github.com/stretchr/testify/require"
)

type sampleApp struct {
	types.BaseApplication
}

func getResponseCallback(t *testing.T, called *bool) ResponseCallback {
	return func (res *types.Response) {
		require.NotNil(t, res)
		*called = true
	}
}

func TestLocalClientCalls(t *testing.T) {
	app := sampleApp{}
	c := NewLocalClient(nil, app)

	gbCalled := false
	c.SetGlobalCallback(func(*types.Request, *types.Response) {
		gbCalled = true
	})

	cbCalled := false
	res := c.EchoAsync("msg", getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)
	require.True(t, gbCalled)

	cbCalled = false
	res = c.FlushAsync(getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.InfoAsync(types.RequestInfo{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.SetOptionAsync(types.RequestSetOption{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.DeliverTxAsync(types.RequestDeliverTx{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.CheckTxAsync(types.RequestCheckTx{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.QueryAsync(types.RequestQuery{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.CommitAsync(getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.InitChainAsync(types.RequestInitChain{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.BeginBlockAsync(types.RequestBeginBlock{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.EndBlockAsync(types.RequestEndBlock{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.BeginRecheckTxAsync(types.RequestBeginRecheckTx{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.EndRecheckTxAsync(types.RequestEndRecheckTx{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.ListSnapshotsAsync(types.RequestListSnapshots{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.OfferSnapshotAsync(types.RequestOfferSnapshot{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.LoadSnapshotChunkAsync(types.RequestLoadSnapshotChunk{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

	cbCalled = false
	res = c.ApplySnapshotChunkAsync(types.RequestApplySnapshotChunk{}, getResponseCallback(t, &cbCalled))
	res.Wait()
	require.True(t, cbCalled)

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

	_, err = c.BeginBlockSync(types.RequestBeginBlock{})
	require.NoError(t, err)

	_, err = c.EndBlockSync(types.RequestEndBlock{})
	require.NoError(t, err)

	_, err = c.BeginRecheckTxSync(types.RequestBeginRecheckTx{})
	require.NoError(t, err)

	_, err = c.EndRecheckTxSync(types.RequestEndRecheckTx{})
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
