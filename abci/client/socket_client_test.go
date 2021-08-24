package abcicli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abcicli "github.com/line/ostracon/abci/client"
	"github.com/line/ostracon/abci/server"
	"github.com/line/ostracon/abci/types"
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
		reqres := c.BeginBlockAsync(types.RequestBeginBlock{}, nil)
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
		reqres := c.BeginBlockAsync(types.RequestBeginBlock{}, nil)
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

func setupClientServer(t *testing.T, app types.Application) (
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
	types.BaseApplication
}

func (slowApp) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
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

type sampleApp struct {
	types.BaseApplication
}

func getResponseCallback(t *testing.T, seq *int, target int) abcicli.ResponseCallback {
	return func (res *types.Response) {
		require.NotNil(t, res)
		require.Equal(t, *seq, target)
		*seq = *seq + 1
	}
}
