package mempool

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/line/ostracon/abci/example/counter"
	"github.com/line/ostracon/abci/example/kvstore"
	abciserver "github.com/line/ostracon/abci/server"
	abci "github.com/line/ostracon/abci/types"
	cfg "github.com/line/ostracon/config"
	"github.com/line/ostracon/libs/log"
	tmrand "github.com/line/ostracon/libs/rand"
	"github.com/line/ostracon/libs/service"
	"github.com/line/ostracon/proxy"
	"github.com/line/ostracon/types"
)

// A cleanupFunc cleans up any config / test files created for a particular
// test.
type cleanupFunc func()

func newMempoolWithApp(cc proxy.ClientCreator) (*CListMempool, cleanupFunc) {
	return newMempoolWithAppAndConfig(cc, cfg.ResetTestRoot("mempool_test"))
}

func newMempoolWithAppAndConfig(cc proxy.ClientCreator, config *cfg.Config) (*CListMempool, cleanupFunc) {
	appConnMem, _ := cc.NewABCIClient()
	appConnMem.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "mempool"))
	err := appConnMem.Start()
	if err != nil {
		panic(err)
	}
	mempool := NewCListMempool(config.Mempool, appConnMem, 0)
	mempool.SetLogger(log.TestingLogger())
	return mempool, func() { os.RemoveAll(config.RootDir) }
}

func ensureNoFire(t *testing.T, ch <-chan struct{}, timeoutMS int) {
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	select {
	case <-ch:
		t.Fatal("Expected not to fire")
	case <-timer.C:
	}
}

func ensureFire(t *testing.T, ch <-chan struct{}, timeoutMS int) {
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	select {
	case <-ch:
	case <-timer.C:
		t.Fatal("Expected to fire")
	}
}

func checkTxs(t *testing.T, mempool Mempool, count int, peerID uint16) types.Txs {
	txs := make(types.Txs, count)
	txInfo := TxInfo{SenderID: peerID}
	for i := 0; i < count; i++ {
		txBytes := make([]byte, 20)
		txs[i] = txBytes
		_, err := rand.Read(txBytes)
		if err != nil {
			t.Error(err)
		}
		if _, err := mempool.CheckTxSync(txBytes, txInfo); err != nil {
			// Skip invalid txs.
			// TestMempoolFilters will fail otherwise. It asserts a number of txs
			// returned.
			if IsPreCheckError(err) {
				continue
			}
			t.Fatalf("CheckTx failed: %v while checking #%d tx", err, i)
		}
	}
	return txs
}

func TestReapMaxBytesMaxGas(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	// Ensure gas calculation behaves as expected
	checkTxs(t, mempool, 1, UnknownPeerID)
	tx0 := mempool.TxsFront().Value.(*mempoolTx)
	// assert that kv store has gas wanted = 1.
	require.Equal(t,
		app.CheckTxSync(abci.RequestCheckTx{Tx: tx0.tx}).GasWanted, int64(1), "KVStore had a gas value neq to 1")
	require.Equal(t, tx0.gasWanted, int64(1), "transactions gas was set incorrectly")
	// ensure each tx is 20 bytes long
	require.Equal(t, len(tx0.tx), 20, "Tx is longer than 20 bytes")
	mempool.Flush()

	// each table driven test creates numTxsToCreate txs with checkTx, and at the end clears all remaining txs.
	// each tx has 20 bytes
	tests := []struct {
		numTxsToCreate int
		maxBytes       int64
		maxGas         int64
		expectedNumTxs int
		maxTxs         int64
	}{
		{20, -1, -1, 20, 0},
		{20, -1, 0, 0, 0},
		{20, -1, 10, 10, 0},
		{20, -1, 30, 20, 0},
		{20, 0, -1, 0, 0},
		{20, 0, 10, 0, 0},
		{20, 10, 10, 0, 0},
		{20, 24, 10, 1, 0},
		{20, 240, 5, 5, 0},
		{20, 240, -1, 10, 0},
		{20, 240, 10, 10, 0},
		{20, 240, 15, 10, 0},
		{20, 20000, -1, 20, 0},
		{20, 20000, 5, 5, 0},
		{20, 20000, 30, 20, 0},
		{20, 20000, 30, 20, 0},
		{20, 20000, 30, 10, 10},
		{20, 20000, 30, 20, 100},
	}
	for tcIndex, tt := range tests {
		checkTxs(t, mempool, tt.numTxsToCreate, UnknownPeerID)
		var got types.Txs
		if tt.maxTxs <= 0 {
			got = mempool.ReapMaxBytesMaxGas(tt.maxBytes, tt.maxGas)
		} else {
			got = mempool.ReapMaxBytesMaxGasMaxTxs(tt.maxBytes, tt.maxGas, tt.maxTxs)
		}
		assert.Equal(t, tt.expectedNumTxs, len(got), "Got %d txs, expected %d, tc #%d",
			len(got), tt.expectedNumTxs, tcIndex)
		mempool.Flush()
	}
}

func TestMempoolFilters(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	emptyTxArr := []types.Tx{[]byte{}}

	nopPreFilter := func(tx types.Tx) error { return nil }

	// each table driven test creates numTxsToCreate txs with checkTx, and at the end clears all remaining txs.
	// each tx has 20 bytes
	tests := []struct {
		numTxsToCreate int
		preFilter      PreCheckFunc
		expectedNumTxs int
	}{
		{10, nopPreFilter, 10},
		{10, PreCheckMaxBytes(10), 0},
		{10, PreCheckMaxBytes(20), 0},
		{10, PreCheckMaxBytes(22), 10},
		{10, PreCheckMaxBytes(30), 10},
	}
	for tcIndex, tt := range tests {
		err := mempool.Update(newTestBlock(1, emptyTxArr),
			abciResponses(len(emptyTxArr), abci.CodeTypeOK), tt.preFilter, nil)
		require.NoError(t, err)
		checkTxs(t, mempool, tt.numTxsToCreate, UnknownPeerID)
		require.Equal(t, tt.expectedNumTxs, mempool.Size(), "mempool had the incorrect size, on test case %d", tcIndex)
		mempool.Flush()
	}
}

func TestMempoolUpdate(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	// 1. Adds valid txs to the cache
	{
		err := mempool.Update(newTestBlock(1, []types.Tx{[]byte{0x01}}),
			abciResponses(1, abci.CodeTypeOK), nil, nil)
		require.NoError(t, err)
		_, err = mempool.CheckTxSync([]byte{0x01}, TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, ErrTxInCache, err)
		}
	}

	// 2. Removes valid txs from the mempool
	{
		_, err := mempool.CheckTxSync([]byte{0x02}, TxInfo{})
		require.NoError(t, err)
		err = mempool.Update(newTestBlock(1, []types.Tx{[]byte{0x02}}), abciResponses(1, abci.CodeTypeOK), nil, nil)
		require.NoError(t, err)
		assert.Zero(t, mempool.Size())
	}

	// 3. Removes invalid transactions from the cache and the mempool (if present)
	{
		_, err := mempool.CheckTxSync([]byte{0x03}, TxInfo{})
		require.NoError(t, err)
		err = mempool.Update(newTestBlock(1, []types.Tx{[]byte{0x03}}), abciResponses(1, 1), nil, nil)
		require.NoError(t, err)
		assert.Zero(t, mempool.Size())

		_, err = mempool.CheckTxSync([]byte{0x03}, TxInfo{})
		require.NoError(t, err)
	}
}

func TestMempool_KeepInvalidTxsInCache(t *testing.T) {
	app := counter.NewApplication(true)
	cc := proxy.NewLocalClientCreator(app)
	wcfg := cfg.DefaultConfig()
	wcfg.Mempool.KeepInvalidTxsInCache = true
	mempool, cleanup := newMempoolWithAppAndConfig(cc, wcfg)
	defer cleanup()

	// 1. An invalid transaction must remain in the cache after Update
	{
		a := make([]byte, 8)
		binary.BigEndian.PutUint64(a, 0)

		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, 1)

		_, err := mempool.CheckTxSync(b, TxInfo{})
		require.NoError(t, err)

		// simulate new block
		_ = app.DeliverTx(abci.RequestDeliverTx{Tx: a})
		_ = app.DeliverTx(abci.RequestDeliverTx{Tx: b})
		err = mempool.Update(newTestBlock(1, []types.Tx{a, b}),
			[]*abci.ResponseDeliverTx{{Code: abci.CodeTypeOK}, {Code: 2}}, nil, nil)
		require.NoError(t, err)

		// a must be added to the cache
		_, err = mempool.CheckTxSync(a, TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, ErrTxInCache, err)
		}

		// b must remain in the cache
		_, err = mempool.CheckTxSync(b, TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, ErrTxInCache, err)
		}
	}

	// 2. An invalid transaction must remain in the cache
	{
		a := make([]byte, 8)
		binary.BigEndian.PutUint64(a, 0)

		// remove a from the cache to test (2)
		mempool.cache.Remove(a)

		_, err := mempool.CheckTxSync(a, TxInfo{})
		require.NoError(t, err)

		_, err = mempool.CheckTxSync(a, TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, ErrTxInCache, err)
		}
	}
}

func TestTxsAvailable(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	mempool.EnableTxsAvailable()

	timeoutMS := 500

	// with no txs, it shouldnt fire
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch of txs, it should only fire once
	txs := checkTxs(t, mempool, 100, UnknownPeerID)
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// call update with half the txs.
	// it should fire once now for the new height
	// since there are still txs left
	committedTxs, txs := txs[:50], txs[50:]
	if err := mempool.Update(newTestBlock(1, committedTxs),
		abciResponses(len(committedTxs), abci.CodeTypeOK), nil, nil); err != nil {
		t.Error(err)
	}
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch more txs. we already fired for this height so it shouldnt fire again
	moreTxs := checkTxs(t, mempool, 50, UnknownPeerID)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// now call update with all the txs. it should not fire as there are no txs left
	committedTxs = append(txs, moreTxs...) // nolint: gocritic
	if err := mempool.Update(newTestBlock(2, committedTxs),
		abciResponses(len(committedTxs), abci.CodeTypeOK), nil, nil); err != nil {
		t.Error(err)
	}
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch more txs, it should only fire once
	checkTxs(t, mempool, 100, UnknownPeerID)
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)
}

func TestSerialReap(t *testing.T) {
	app := counter.NewApplication(true)
	app.SetOption(abci.RequestSetOption{Key: "serial", Value: "on"})
	cc := proxy.NewLocalClientCreator(app)

	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	appConnCon, _ := cc.NewABCIClient()
	appConnCon.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "consensus"))
	err := appConnCon.Start()
	require.Nil(t, err)

	cacheMap := make(map[string]struct{})
	deliverTxsRange := func(start, end int) {
		// Deliver some txs.
		for i := start; i < end; i++ {

			// This will succeed
			txBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(txBytes, uint64(i))
			_, err := mempool.CheckTxSync(txBytes, TxInfo{})
			_, cached := cacheMap[string(txBytes)]
			if cached {
				require.NotNil(t, err, "expected error for cached tx")
			} else {
				require.Nil(t, err, "expected no err for uncached tx")
			}
			cacheMap[string(txBytes)] = struct{}{}

			// Duplicates are cached and should return error
			_, err = mempool.CheckTxSync(txBytes, TxInfo{})
			require.NotNil(t, err, "Expected error after CheckTx on duplicated tx")
		}
	}

	reapCheck := func(exp int) {
		txs := mempool.ReapMaxBytesMaxGas(-1, -1)
		require.Equal(t, len(txs), exp, fmt.Sprintf("Expected to reap %v txs but got %v", exp, len(txs)))
	}

	updateRange := func(start, end int) {
		txs := make([]types.Tx, 0)
		for i := start; i < end; i++ {
			txBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(txBytes, uint64(i))
			txs = append(txs, txBytes)
		}
		if err := mempool.Update(newTestBlock(0, txs),
			abciResponses(len(txs), abci.CodeTypeOK), nil, nil); err != nil {
			t.Error(err)
		}
	}

	commitRange := func(start, end int) {
		// Deliver some txs.
		for i := start; i < end; i++ {
			txBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(txBytes, uint64(i))
			res, err := appConnCon.DeliverTxSync(abci.RequestDeliverTx{Tx: txBytes})
			if err != nil {
				t.Errorf("client error committing tx: %v", err)
			}
			if res.IsErr() {
				t.Errorf("error committing tx. Code:%v result:%X log:%v",
					res.Code, res.Data, res.Log)
			}
		}
		res, err := appConnCon.CommitSync()
		if err != nil {
			t.Errorf("client error committing: %v", err)
		}
		if len(res.Data) != 8 {
			t.Errorf("error committing. Hash:%X", res.Data)
		}
	}

	// ----------------------------------------

	// Deliver some txs.
	deliverTxsRange(0, 100)

	// Reap the txs.
	reapCheck(100)

	// Reap again.  We should get the same amount
	reapCheck(100)

	// Deliver 0 to 999, we should reap 900 new txs
	// because 100 were already counted.
	deliverTxsRange(0, 1000)

	// Reap the txs.
	reapCheck(1000)

	// Reap again.  We should get the same amount
	reapCheck(1000)

	// Commit from the conensus AppConn
	commitRange(0, 500)
	updateRange(0, 500)

	// We should have 500 left.
	reapCheck(500)

	// Deliver 100 invalid txs and 100 valid txs
	deliverTxsRange(900, 1100)

	// We should have 600 now.
	reapCheck(600)
}

func TestMempoolCloseWAL(t *testing.T) {
	// 1. Create the temporary directory for mempool and WAL testing.
	rootDir, err := ioutil.TempDir("", "mempool-test")
	require.Nil(t, err, "expecting successful tmpdir creation")

	// 2. Ensure that it doesn't contain any elements -- Sanity check
	m1, err := filepath.Glob(filepath.Join(rootDir, "*"))
	require.Nil(t, err, "successful globbing expected")
	require.Equal(t, 0, len(m1), "no matches yet")

	// 3. Create the mempool
	wcfg := cfg.DefaultConfig()
	wcfg.Mempool.RootDir = rootDir
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithAppAndConfig(cc, wcfg)
	defer cleanup()
	mempool.height = 10
	err = mempool.InitWAL()
	require.NoError(t, err)

	// 4. Ensure that the directory contains the WAL file
	m2, err := filepath.Glob(filepath.Join(rootDir, "*"))
	require.Nil(t, err, "successful globbing expected")
	require.Equal(t, 1, len(m2), "expecting the wal match in")

	// 5. Write some contents to the WAL
	_, err = mempool.CheckTxSync(types.Tx([]byte("foo")), TxInfo{})
	require.NoError(t, err)
	walFilepath := mempool.wal.Path
	sum1 := checksumFile(walFilepath, t)

	// 6. Sanity check to ensure that the written TX matches the expectation.
	require.Equal(t, sum1, checksumIt([]byte("foo\n")), "foo with a newline should be written")

	// 7. Invoke CloseWAL() and ensure it discards the
	// WAL thus any other write won't go through.
	mempool.CloseWAL()
	_, err = mempool.CheckTxSync(types.Tx([]byte("bar")), TxInfo{})
	require.NoError(t, err)
	sum2 := checksumFile(walFilepath, t)
	require.Equal(t, sum1, sum2, "expected no change to the WAL after invoking CloseWAL() since it was discarded")

	// 8. Sanity check to ensure that the WAL file still exists
	m3, err := filepath.Glob(filepath.Join(rootDir, "*"))
	require.Nil(t, err, "successful globbing expected")
	require.Equal(t, 1, len(m3), "expecting the wal match in")
}

func TestMempool_CheckTxChecksTxSize(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempl, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	maxTxSize := mempl.config.MaxTxBytes

	testCases := []struct {
		len int
		err bool
	}{
		// check small txs. no error
		0: {10, false},
		1: {1000, false},
		2: {1000000, false},

		// check around maxTxSize
		3: {maxTxSize - 1, false},
		4: {maxTxSize, false},
		5: {maxTxSize + 1, true},
	}

	for i, testCase := range testCases {
		caseString := fmt.Sprintf("case %d, len %d", i, testCase.len)

		tx := tmrand.Bytes(testCase.len)

		_, err := mempl.CheckTxSync(tx, TxInfo{})
		bv := gogotypes.BytesValue{Value: tx}
		bz, err2 := bv.Marshal()
		require.NoError(t, err2)
		require.Equal(t, len(bz), proto.Size(&bv), caseString)

		if !testCase.err {
			require.NoError(t, err, caseString)
		} else {
			require.Equal(t, err, ErrTxTooLarge{maxTxSize, testCase.len}, caseString)
		}
	}
}

func TestMempoolTxsBytes(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	config.Mempool.MaxTxsBytes = 10
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	// 1. zero by default
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 2. len(tx) after CheckTx
	_, err := mempool.CheckTxSync([]byte{0x01}, TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, mempool.TxsBytes())

	// 3. zero again after tx is removed by Update
	err = mempool.Update(newTestBlock(1, []types.Tx{[]byte{0x01}}),
		abciResponses(1, abci.CodeTypeOK), nil, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 4. zero after Flush
	_, err = mempool.CheckTxSync([]byte{0x02, 0x03}, TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, mempool.TxsBytes())

	mempool.Flush()
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 5. ErrMempoolIsFull is returned when/if MaxTxsBytes limit is reached.
	_, err = mempool.CheckTxSync([]byte{0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04}, TxInfo{})
	require.NoError(t, err)
	_, err = mempool.CheckTxSync([]byte{0x05}, TxInfo{})
	if assert.Error(t, err) {
		assert.IsType(t, ErrMempoolIsFull{}, err)
	}

	// 6. zero after tx is rechecked and removed due to not being valid anymore
	app2 := counter.NewApplication(true)
	cc = proxy.NewLocalClientCreator(app2)
	mempool, cleanup = newMempoolWithApp(cc)
	defer cleanup()

	txBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(txBytes, uint64(0))

	_, err = mempool.CheckTxSync(txBytes, TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 8, mempool.TxsBytes())

	appConnCon, _ := cc.NewABCIClient()
	appConnCon.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "consensus"))
	err = appConnCon.Start()
	require.Nil(t, err)
	t.Cleanup(func() {
		if err := appConnCon.Stop(); err != nil {
			t.Error(err)
		}
	})
	res, err := appConnCon.DeliverTxSync(abci.RequestDeliverTx{Tx: txBytes})
	require.NoError(t, err)
	require.EqualValues(t, 0, res.Code)
	res2, err := appConnCon.CommitSync()
	require.NoError(t, err)
	require.NotEmpty(t, res2.Data)

	// Pretend like we committed nothing so txBytes gets rechecked and removed.
	err = mempool.Update(newTestBlock(1, []types.Tx{}), abciResponses(0, abci.CodeTypeOK), nil, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 7. Test RemoveTxByKey function
	_, err = mempool.CheckTxSync([]byte{0x06}, TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, mempool.TxsBytes())
	mempool.RemoveTxByKey(TxKey([]byte{0x07}), true)
	assert.EqualValues(t, 1, mempool.TxsBytes())
	mempool.RemoveTxByKey(TxKey([]byte{0x06}), true)
	assert.EqualValues(t, 0, mempool.TxsBytes())

}

// This will non-deterministically catch some concurrency failures like
// https://github.com/tendermint/tendermint/issues/3509
// TODO: all of the tests should probably also run using the remote proxy app
// since otherwise we're not actually testing the concurrency of the mempool here!
func TestMempoolRemoteAppConcurrency(t *testing.T) {
	sockPath := fmt.Sprintf("unix:///tmp/echo_%v.sock", tmrand.Str(6))
	app := kvstore.NewApplication()
	cc, server := newRemoteApp(t, sockPath, app)
	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Error(err)
		}
	})
	config := cfg.ResetTestRoot("mempool_test")
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	// generate small number of txs
	nTxs := 10
	txLen := 200
	txs := make([]types.Tx, nTxs)
	for i := 0; i < nTxs; i++ {
		txs[i] = tmrand.Bytes(txLen)
	}

	// simulate a group of peers sending them over and over
	N := config.Mempool.Size
	maxPeers := 5
	for i := 0; i < N; i++ {
		peerID := mrand.Intn(maxPeers)
		txNum := mrand.Intn(nTxs)
		tx := txs[txNum]

		// this will err with ErrTxInCache many times ...
		mempool.CheckTxSync(tx, TxInfo{SenderID: uint16(peerID)}) // nolint: errcheck
	}
	err := mempool.FlushAppConn()
	require.NoError(t, err)
}

func newTestBlock(height int64, txs types.Txs) *types.Block {
	return &types.Block{
		Header: types.Header{
			Height: height,
		},
		Data: types.Data{
			Txs: txs,
		},
	}
}

// caller must close server
func newRemoteApp(
	t *testing.T,
	addr string,
	app abci.Application,
) (
	clientCreator proxy.ClientCreator,
	server service.Service,
) {
	clientCreator = proxy.NewRemoteClientCreator(addr, "socket", true)

	// Start server
	server = abciserver.NewSocketServer(addr, app)
	server.SetLogger(log.TestingLogger().With("module", "abci-server"))
	if err := server.Start(); err != nil {
		t.Fatalf("Error starting socket server: %v", err.Error())
	}
	return clientCreator, server
}

func checksumIt(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func checksumFile(p string, t *testing.T) string {
	data, err := ioutil.ReadFile(p)
	require.Nil(t, err, "expecting successful read of %q", p)
	return checksumIt(data)
}

func abciResponses(n int, code uint32) []*abci.ResponseDeliverTx {
	responses := make([]*abci.ResponseDeliverTx, 0, n)
	for i := 0; i < n; i++ {
		responses = append(responses, &abci.ResponseDeliverTx{Code: code})
	}
	return responses
}

func TestTxMempoolPostCheckError(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{
			name: "error",
			err:  errors.New("test error"),
		},
		{
			name: "no error",
			err:  nil,
		},
	}
	for _, tc := range cases {
		testCase := tc
		t.Run(testCase.name, func(t *testing.T) {
			app := kvstore.NewApplication()
			cc := proxy.NewLocalClientCreator(app)
			mempool, cleanup := newMempoolWithApp(cc)
			defer cleanup()

			mempool.postCheck = func(_ types.Tx, _ *abci.ResponseCheckTx) error {
				return testCase.err
			}

			tx := types.Tx{1}
			_, err := mempool.CheckTxSync(tx, TxInfo{})
			require.NoError(t, err)

			req := abci.RequestCheckTx{
				Tx:   tx,
				Type: abci.CheckTxType_Recheck,
			}
			res := &abci.Response{}

			m := sync.Mutex{}
			m.Lock()
			mempool.proxyAppConn.CheckTxAsync(req, func(r *abci.Response) {
				res = r
				m.Unlock()
			})

			checkTxRes, ok := res.Value.(*abci.Response_CheckTx)
			require.True(t, ok)
			expectedErrString := ""
			if testCase.err != nil {
				expectedErrString = testCase.err.Error()
			}
			require.Equal(t, expectedErrString, checkTxRes.CheckTx.MempoolError)
		})
	}
}
