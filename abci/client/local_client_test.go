package abcicli

import (
	"testing"
	"time"

	"github.com/line/ostracon/abci/types"
	"github.com/stretchr/testify/require"
)

type sampleApp struct {
	types.BaseApplication
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

func TestLocalClientCalls(t *testing.T) {
	app := sampleApp{}
	c := NewLocalClient(nil, app)

	c.SetResponseCallback(func(*types.Request, *types.Response) {
	})

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
