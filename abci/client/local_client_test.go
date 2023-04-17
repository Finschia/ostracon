package abcicli

import (
	"testing"
	"time"

	"github.com/tendermint/tendermint/abci/types"

	ocabci "github.com/Finschia/ostracon/abci/types"
	"github.com/stretchr/testify/require"
)

type sampleApp struct {
	ocabci.BaseApplication
}

func newDoneChan(t *testing.T) chan struct{} {
	result := make(chan struct{})
	go func() {
		select {
		case <-time.After(time.Second):
			require.Fail(t, "callback is not called for a second")
		case <-result:
			return
		}
	}()
	return result
}

func getResponseCallback(t *testing.T) ResponseCallback {
	doneChan := newDoneChan(t)
	return func(res *ocabci.Response) {
		require.NotNil(t, res)
		doneChan <- struct{}{}
	}
}

func TestLocalClientCalls(t *testing.T) {
	app := sampleApp{}
	c := NewLocalClient(nil, app)

	c.SetGlobalCallback(func(*ocabci.Request, *ocabci.Response) {
	})

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
