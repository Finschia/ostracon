package abcicli

import (
	"testing"

	"github.com/line/ostracon/abci/types"
	"github.com/stretchr/testify/require"
)

type sampleApp struct {
	types.BaseApplication
}

func getResponseCallback(t *testing.T, seq *int, target int) ResponseCallback {
	return func (res *types.Response) {
		require.NotNil(t, res)
		require.Equal(t, *seq, target)
		*seq = *seq + 1
	}
}

func TestLocalClientCalls(t *testing.T) {
	app := sampleApp{}
	c := NewLocalClient(nil, app)

	gbCalled := false
	c.SetGlobalCallback(func(*types.Request, *types.Response) {
		gbCalled = true
	})

	callSeq := 0
	res := c.EchoAsync("msg", getResponseCallback(t, &callSeq, 0))
	res.Wait()
	// callback is not called synchronously, so we cannot assure that the callback is called
	// But we can assure the callback is called by turns
	require.True(t, gbCalled)

	res = c.FlushAsync(getResponseCallback(t, &callSeq, 1))
	res.Wait()

	res = c.InfoAsync(types.RequestInfo{}, getResponseCallback(t, &callSeq, 2))
	res.Wait()

	res = c.SetOptionAsync(types.RequestSetOption{}, getResponseCallback(t, &callSeq, 3))
	res.Wait()

	res = c.DeliverTxAsync(types.RequestDeliverTx{}, getResponseCallback(t, &callSeq, 4))
	res.Wait()

	res = c.CheckTxAsync(types.RequestCheckTx{}, getResponseCallback(t, &callSeq, 5))
	res.Wait()

	res = c.QueryAsync(types.RequestQuery{}, getResponseCallback(t, &callSeq, 6))
	res.Wait()

	res = c.CommitAsync(getResponseCallback(t, &callSeq, 7))
	res.Wait()

	res = c.InitChainAsync(types.RequestInitChain{}, getResponseCallback(t, &callSeq, 8))
	res.Wait()

	res = c.BeginBlockAsync(types.RequestBeginBlock{}, getResponseCallback(t, &callSeq, 9))
	res.Wait()

	res = c.EndBlockAsync(types.RequestEndBlock{}, getResponseCallback(t, &callSeq, 10))
	res.Wait()

	res = c.BeginRecheckTxAsync(types.RequestBeginRecheckTx{}, getResponseCallback(t, &callSeq, 11))
	res.Wait()

	res = c.EndRecheckTxAsync(types.RequestEndRecheckTx{}, getResponseCallback(t, &callSeq, 12))
	res.Wait()

	res = c.ListSnapshotsAsync(types.RequestListSnapshots{}, getResponseCallback(t, &callSeq, 13))
	res.Wait()

	res = c.OfferSnapshotAsync(types.RequestOfferSnapshot{}, getResponseCallback(t, &callSeq, 14))
	res.Wait()

	res = c.LoadSnapshotChunkAsync(types.RequestLoadSnapshotChunk{}, getResponseCallback(t, &callSeq, 15))
	res.Wait()

	res = c.ApplySnapshotChunkAsync(types.RequestApplySnapshotChunk{}, getResponseCallback(t, &callSeq, 16))
	res.Wait()

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
