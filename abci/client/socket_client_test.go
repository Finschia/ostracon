package abcicli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmabci "github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/line/ostracon/abci/client"
	"github.com/line/ostracon/abci/server"
	abci "github.com/line/ostracon/abci/types"
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
		reqres := c.BeginBlockAsync(abci.RequestBeginBlock{}, nil)
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
		reqres := c.BeginBlockAsync(abci.RequestBeginBlock{}, nil)
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

func setupClientServer(t *testing.T, app abci.Application) (
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
	abci.BaseApplication
}

func (slowApp) BeginBlock(req abci.RequestBeginBlock) tmabci.ResponseBeginBlock {
	time.Sleep(200 * time.Millisecond)
	return tmabci.ResponseBeginBlock{}
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

	c.SetGlobalCallback(func(*abci.Request, *abci.Response) {
	})

	c.EchoAsync("msg", getResponseCallback(t))
	c.FlushAsync(getResponseCallback(t))
	c.InfoAsync(tmabci.RequestInfo{}, getResponseCallback(t))
	c.SetOptionAsync(tmabci.RequestSetOption{}, getResponseCallback(t))
	c.DeliverTxAsync(tmabci.RequestDeliverTx{}, getResponseCallback(t))
	c.CheckTxAsync(tmabci.RequestCheckTx{}, getResponseCallback(t))
	c.QueryAsync(tmabci.RequestQuery{}, getResponseCallback(t))
	c.CommitAsync(getResponseCallback(t))
	c.InitChainAsync(abci.RequestInitChain{}, getResponseCallback(t))
	c.BeginBlockAsync(abci.RequestBeginBlock{}, getResponseCallback(t))
	c.EndBlockAsync(tmabci.RequestEndBlock{}, getResponseCallback(t))
	c.BeginRecheckTxAsync(abci.RequestBeginRecheckTx{}, getResponseCallback(t))
	c.EndRecheckTxAsync(abci.RequestEndRecheckTx{}, getResponseCallback(t))
	c.ListSnapshotsAsync(tmabci.RequestListSnapshots{}, getResponseCallback(t))
	c.OfferSnapshotAsync(tmabci.RequestOfferSnapshot{}, getResponseCallback(t))
	c.LoadSnapshotChunkAsync(tmabci.RequestLoadSnapshotChunk{}, getResponseCallback(t))
	c.ApplySnapshotChunkAsync(tmabci.RequestApplySnapshotChunk{}, getResponseCallback(t))

	_, err := c.EchoSync("msg")
	require.NoError(t, err)

	_, err = c.FlushSync()
	require.NoError(t, err)

	_, err = c.InfoSync(tmabci.RequestInfo{})
	require.NoError(t, err)

	_, err = c.SetOptionSync(tmabci.RequestSetOption{})
	require.NoError(t, err)

	_, err = c.DeliverTxSync(tmabci.RequestDeliverTx{})
	require.NoError(t, err)

	_, err = c.CheckTxSync(tmabci.RequestCheckTx{})
	require.NoError(t, err)

	_, err = c.QuerySync(tmabci.RequestQuery{})
	require.NoError(t, err)

	_, err = c.CommitSync()
	require.NoError(t, err)

	_, err = c.InitChainSync(abci.RequestInitChain{})
	require.NoError(t, err)

	_, err = c.BeginBlockSync(abci.RequestBeginBlock{})
	require.NoError(t, err)

	_, err = c.EndBlockSync(tmabci.RequestEndBlock{})
	require.NoError(t, err)

	_, err = c.BeginRecheckTxSync(abci.RequestBeginRecheckTx{})
	require.NoError(t, err)

	_, err = c.EndRecheckTxSync(abci.RequestEndRecheckTx{})
	require.NoError(t, err)

	_, err = c.ListSnapshotsSync(tmabci.RequestListSnapshots{})
	require.NoError(t, err)

	_, err = c.OfferSnapshotSync(tmabci.RequestOfferSnapshot{})
	require.NoError(t, err)

	_, err = c.LoadSnapshotChunkSync(tmabci.RequestLoadSnapshotChunk{})
	require.NoError(t, err)

	_, err = c.ApplySnapshotChunkSync(tmabci.RequestApplySnapshotChunk{})
	require.NoError(t, err)
}

type sampleApp struct {
	abci.BaseApplication
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
	return func(res *abci.Response) {
		require.NotNil(t, res)
		doneChan <- struct{}{}
	}
}
