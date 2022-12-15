package mempool

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/line/ostracon/abci/example/counter"
	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/config"
	"github.com/line/ostracon/libs/log"
	tmversion "github.com/line/ostracon/proto/ostracon/version"
	"github.com/line/ostracon/proxy"
	"github.com/line/ostracon/types"
	"github.com/line/ostracon/version"
	"github.com/stretchr/testify/require"
)

func setupCListMempool(ctx context.Context, t testing.TB,
	height int64, size, cacheSize int) *CListMempool {
	t.Helper()

	var cancel context.CancelFunc
	_, cancel = context.WithCancel(ctx)

	cfg := config.ResetTestRoot(strings.ReplaceAll(t.Name(), "/", "|"))
	cfg.Mempool = config.DefaultMempoolConfig()
	logLevel, _ := log.AllowLevel("info")
	logger := log.NewFilter(log.NewOCLogger(log.NewSyncWriter(os.Stdout)), logLevel)

	appConn := proxy.NewAppConns(proxy.NewLocalClientCreator(counter.NewApplication(false)))
	require.NoError(t, appConn.Start())

	t.Cleanup(func() {
		os.RemoveAll(cfg.RootDir)
		cancel()
		appConn.Stop() // nolint: errcheck // ignore
	})

	if size > -1 {
		cfg.Mempool.Size = size
	}
	if cacheSize > -1 {
		cfg.Mempool.CacheSize = cacheSize
	}
	mem := NewCListMempool(cfg.Mempool, appConn.Mempool(), height)
	mem.SetLogger(logger)
	return mem
}

func TestCListMempool_SystemTestWithCacheSizeDefault(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mem := setupCListMempool(ctx, t, 1, -1, -1) // size=5000, cacheSize=10000
	recvTxCnt := &receiveTxCounter{}
	stop := make(chan struct{}, 1)
	go gossipRoutine(ctx, t, mem, recvTxCnt, stop)
	makeBlocksAndCommits(ctx, t, mem)
	close(stop)

	// check the inconsistency
	require.Equal(t, mem.txs.Len(), sizeOfSyncMap(&mem.txsMap))

	expected := int64(0)
	actual := recvTxCnt.threadSafeCopy()
	assert.NotEqual(t, expected, actual.sent, fmt.Sprintf("actual %d", actual.sent))
	assert.NotEqual(t, expected, actual.success, fmt.Sprintf("actual %d", actual.success))
	assert.NotEqual(t, expected, actual.failInMap, fmt.Sprintf("actual %d", actual.failInMap))
	assert.NotEqual(t, expected, actual.failInCache, fmt.Sprintf("actual %d", actual.failInCache))
	assert.Equal(t, expected, actual.failTooLarge)
	assert.NotEqual(t, expected, actual.failIsFull, fmt.Sprintf("actual %d", actual.failIsFull))
	assert.Equal(t, expected, actual.failPreCheck)
	assert.Equal(t, expected, actual.abciFail)
}

func sizeOfSyncMap(m *sync.Map) int {
	length := 0
	m.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

func createProposalBlockAndDeliverTxs(
	mem *CListMempool, height int64) (*types.Block, []*abci.ResponseDeliverTx) {
	// mempool.lock/unlock in ReapMaxBytesMaxGasMaxTxs
	txs := mem.ReapMaxBytesMaxGasMaxTxs(mem.config.MaxTxsBytes, 0, int64(mem.config.Size))
	block := types.MakeBlock(height, txs, nil, nil, tmversion.Consensus{
		Block: version.BlockProtocol,
		App:   version.AppProtocol,
	})
	deliverTxResponses := make([]*abci.ResponseDeliverTx, len(block.Txs))
	for i, tx := range block.Txs {
		deliverTxResponses[i] = &abci.ResponseDeliverTx{
			Code: abci.CodeTypeOK,
			Data: tx,
		}
	}
	return block, deliverTxResponses
}

func commitBlock(ctx context.Context, t *testing.T,
	mem *CListMempool, block *types.Block, deliverTxResponses []*abci.ResponseDeliverTx) {
	mem.Lock()
	defer mem.Unlock()
	err := mem.Update(block, deliverTxResponses, nil, nil)
	require.NoError(t, err)
}

func receiveTx(ctx context.Context, t *testing.T,
	mem *CListMempool, tx []byte, receiveTxCounter *receiveTxCounter) {
	atomic.AddInt64(&receiveTxCounter.sent, 1)
	txInfo := TxInfo{}
	// mempool.lock/unlock in CheckTxAsync
	mem.CheckTxAsync(tx, txInfo,
		func(err error) {
			if err != nil {
				switch err {
				case ErrTxInCache:
					atomic.AddInt64(&receiveTxCounter.failInCache, 1)
				case ErrTxInMap:
					atomic.AddInt64(&receiveTxCounter.failInMap, 1)
				}
				switch err.(type) {
				case ErrTxTooLarge:
					atomic.AddInt64(&receiveTxCounter.failTooLarge, 1)
				case ErrMempoolIsFull:
					atomic.AddInt64(&receiveTxCounter.failIsFull, 1)
				case ErrPreCheck:
					atomic.AddInt64(&receiveTxCounter.failPreCheck, 1)
				}
			}
		},
		func(res *abci.Response) {
			resCheckTx := res.GetCheckTx()
			if resCheckTx.Code != abci.CodeTypeOK && len(resCheckTx.Log) != 0 {
				atomic.AddInt64(&receiveTxCounter.abciFail, 1)
			} else {
				atomic.AddInt64(&receiveTxCounter.success, 1)
			}
		})
}

type receiveTxCounter struct {
	sent         int64
	success      int64
	failInMap    int64
	failInCache  int64
	failTooLarge int64
	failIsFull   int64
	failPreCheck int64
	abciFail     int64
}

func (r *receiveTxCounter) threadSafeCopy() receiveTxCounter {
	return receiveTxCounter{
		sent:         atomic.LoadInt64(&r.sent),
		success:      atomic.LoadInt64(&r.success),
		failInMap:    atomic.LoadInt64(&r.failInMap),
		failInCache:  atomic.LoadInt64(&r.failInCache),
		failTooLarge: atomic.LoadInt64(&r.failTooLarge),
		failIsFull:   atomic.LoadInt64(&r.failIsFull),
		failPreCheck: atomic.LoadInt64(&r.failPreCheck),
		abciFail:     atomic.LoadInt64(&r.abciFail),
	}
}

func gossipRoutine(ctx context.Context, t *testing.T, mem *CListMempool,
	receiveTxCounter *receiveTxCounter, stop chan struct{}) {
	for i := 0; i < nodeNum; i++ {
		select {
		case <-stop:
			return
		default:
			go receiveRoutine(ctx, t, mem, receiveTxCounter, stop)
		}
	}
}

func receiveRoutine(ctx context.Context, t *testing.T, mem *CListMempool,
	receiveTxCounter *receiveTxCounter, stop chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
			tx := []byte(strconv.Itoa(rand.Intn(mem.config.CacheSize * 2)))
			// mempool.lock/unlock in CheckTxAsync
			receiveTx(ctx, t, mem, tx, receiveTxCounter)
			if receiveTxCounter.sent%2000 == 0 {
				time.Sleep(time.Second) // for avoiding mempool full
			}
		}
	}
}

func makeBlocksAndCommits(ctx context.Context, t *testing.T, mem *CListMempool) {
	for i := 0; i < blockNum; i++ {
		block, deliverTxResponses := createProposalBlockAndDeliverTxs(mem, int64(i+1))
		time.Sleep(randQuadraticCurveInterval(deliveredTimeMin, deliveredTimeMax, deliveredTimeRadix))
		commitBlock(ctx, t, mem, block, deliverTxResponses)
		time.Sleep(randQuadraticCurveInterval(blockIntervalMin, blockIntervalMax, blockIntervalRadix))
	}
}

const (
	nodeNum            = 1
	blockNum           = 10
	blockIntervalMin   = 1.0 // second
	blockIntervalMax   = 1.0 // second
	blockIntervalRadix = 0.1
	deliveredTimeMin   = 2.0  // second
	deliveredTimeMax   = 10.0 // second
	deliveredTimeRadix = 0.1
)

func randQuadraticCurveInterval(min, max, radix float64) time.Duration {
	rand.Seed(time.Now().UnixNano())
	x := rand.Float64()*(max-min) + min
	y := (x * x) * radix
	return time.Duration(y*1000) * time.Millisecond
}
