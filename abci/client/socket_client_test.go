package abcicli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/line/ostracon/abci/client"
	"github.com/line/ostracon/abci/server"
	ocabci "github.com/line/ostracon/abci/types"
	tmrand "github.com/line/ostracon/libs/rand"
	"github.com/line/ostracon/libs/service"
)

func TestProperSyncCalls(t *testing.T) {
	app := slowApp{}

	s, c := setupClientServer(t, app)
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Error(err)
		}
	})
	t.Cleanup(func() {
		if err := c.Stop(); err != nil {
			t.Error(err)
		}
	})

	resp := make(chan error, 1)
	go func() {
		// This is BeginBlockSync unrolled....
		reqres := c.BeginBlockAsync(ocabci.RequestBeginBlock{}, nil)
		_, err := c.FlushSync()
		require.NoError(t, err)
		res := reqres.Response.GetBeginBlock()
		require.NotNil(t, res)
		resp <- c.Error()
	}()

	select {
	case <-time.After(time.Second):
		require.Fail(t, "No response arrived")
	case err, ok := <-resp:
		require.True(t, ok, "Must not close channel")
		assert.NoError(t, err, "This should return success")
	}
}

func TestHangingSyncCalls(t *testing.T) {
	app := slowApp{}

	s, c := setupClientServer(t, app)
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Log(err)
		}
	})
	t.Cleanup(func() {
		if err := c.Stop(); err != nil {
			t.Log(err)
		}
	})

	resp := make(chan error, 1)
	go func() {
		// Start BeginBlock and flush it
		reqres := c.BeginBlockAsync(ocabci.RequestBeginBlock{}, nil)
		flush := c.FlushAsync(nil)
		// wait 20 ms for all events to travel socket, but
		// no response yet from server
		time.Sleep(20 * time.Millisecond)
		// kill the server, so the connections break
		err := s.Stop()
		require.NoError(t, err)

		// wait for the response from BeginBlock
		reqres.Wait()
		flush.Wait()
		resp <- c.Error()
	}()

	select {
	case <-time.After(time.Second):
		require.Fail(t, "No response arrived")
	case err, ok := <-resp:
		require.True(t, ok, "Must not close channel")
		assert.Error(t, err, "We should get EOF error")
	}
}

func setupClientServer(t *testing.T, app ocabci.Application) (
	service.Service, abcicli.Client) {
	// some port between 20k and 30k
	port := 20000 + tmrand.Int32()%10000
	addr := fmt.Sprintf("localhost:%d", port)

	s, err := server.NewServer(addr, "socket", app)
	require.NoError(t, err)
	err = s.Start()
	require.NoError(t, err)

	c := abcicli.NewSocketClient(addr, true)
	err = c.Start()
	require.NoError(t, err)

	return s, c
}

type slowApp struct {
	ocabci.BaseApplication
}

func (slowApp) BeginBlock(req ocabci.RequestBeginBlock) types.ResponseBeginBlock {
	time.Sleep(200 * time.Millisecond)
	return types.ResponseBeginBlock{}
}

func TestSockerClientCalls(t *testing.T) {
	app := sampleApp{}

	s, c := setupClientServer(t, app)
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Error(err)
		}
	})
	t.Cleanup(func() {
		if err := c.Stop(); err != nil {
			t.Error(err)
		}
	})

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

func getResponseCallback(t *testing.T) abcicli.ResponseCallback {
	doneChan := newDoneChan(t)
	return func(res *ocabci.Response) {
		require.NotNil(t, res)
		doneChan <- struct{}{}
	}
}
