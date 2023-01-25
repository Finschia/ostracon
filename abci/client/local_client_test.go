package abcicli

import (
	"testing"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"

	ocabci "github.com/line/ostracon/abci/types"
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
	c.InfoAsync(abci.RequestInfo{}, getResponseCallback(t))
	c.SetOptionAsync(abci.RequestSetOption{}, getResponseCallback(t))
	c.DeliverTxAsync(abci.RequestDeliverTx{}, getResponseCallback(t))
	c.CheckTxAsync(abci.RequestCheckTx{}, getResponseCallback(t))
	c.QueryAsync(abci.RequestQuery{}, getResponseCallback(t))
	c.CommitAsync(getResponseCallback(t))
	c.InitChainAsync(abci.RequestInitChain{}, getResponseCallback(t))
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

	_, err = c.InitChainSync(abci.RequestInitChain{})
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
