package mempool

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	abci "github.com/line/ostracon/abci/types"
	cfg "github.com/line/ostracon/config"
	auto "github.com/line/ostracon/libs/autofile"
	"github.com/line/ostracon/libs/clist"
	"github.com/line/ostracon/libs/log"
	tmmath "github.com/line/ostracon/libs/math"
	tmos "github.com/line/ostracon/libs/os"
	tmsync "github.com/line/ostracon/libs/sync"
	"github.com/line/ostracon/p2p"
	"github.com/line/ostracon/proxy"
	"github.com/line/ostracon/types"
)

// TxKeySize is the size of the transaction key index
const TxKeySize = sha256.Size

var newline = []byte("\n")

//--------------------------------------------------------------------------------

// CListMempool is an ordered in-memory pool for transactions before they are
// proposed in a consensus round. Transaction validity is checked using the
// CheckTx abci message before the transaction is added to the pool. The
// mempool uses a concurrent list structure for storing transactions that can
// be efficiently accessed by multiple concurrent readers.
type CListMempool struct {
	// Atomic integers
	height   int64 // the last block Update()'d to
	txsBytes int64 // total size of mempool, in bytes

	reserved      int   // the number of checking tx and it should be considered when checking mempool full
	reservedBytes int64 // size of checking tx and it should be considered when checking mempool full
	reservedMtx   sync.Mutex

	// notify listeners (ie. consensus) when txs are available
	notifiedTxsAvailable bool
	txsAvailable         chan struct{} // fires once for each height, when the mempool is not empty

	config *cfg.MempoolConfig

	// Exclusive mutex for Update method to prevent concurrent execution of
	// CheckTx or ReapMaxBytesMaxGas(ReapMaxTxs) methods.
	updateMtx tmsync.RWMutex
	preCheck  PreCheckFunc

	chReqCheckTx chan *requestCheckTxAsync

	postCheck PostCheckFunc

	wal          *auto.AutoFile // a log of mempool txs
	txs          *clist.CList   // concurrent linked-list of good txs
	proxyAppConn proxy.AppConnMempool

	// Map for quick access to txs to record sender in CheckTx.
	// txsMap: txKey -> CElement
	txsMap sync.Map

	// Keep a cache of already-seen txs.
	// This reduces the pressure on the proxyApp.
	cache txCache

	logger log.Logger

	metrics *Metrics
}

type requestCheckTxAsync struct {
	tx        types.Tx
	txInfo    TxInfo
	prepareCb func(error)
	checkTxCb func(*abci.Response)
}

var _ Mempool = &CListMempool{}

// CListMempoolOption sets an optional parameter on the mempool.
type CListMempoolOption func(*CListMempool)

// NewCListMempool returns a new mempool with the given configuration and connection to an application.
func NewCListMempool(
	config *cfg.MempoolConfig,
	proxyAppConn proxy.AppConnMempool,
	height int64,
	options ...CListMempoolOption,
) *CListMempool {
	mempool := &CListMempool{
		config:       config,
		proxyAppConn: proxyAppConn,
		txs:          clist.New(),
		height:       height,
		chReqCheckTx: make(chan *requestCheckTxAsync, config.Size),
		logger:       log.NewNopLogger(),
		metrics:      NopMetrics(),
	}
	if config.CacheSize > 0 {
		mempool.cache = newMapTxCache(config.CacheSize)
	} else {
		mempool.cache = nopTxCache{}
	}
	proxyAppConn.SetGlobalCallback(mempool.globalCb)
	for _, option := range options {
		option(mempool)
	}
	go mempool.checkTxAsyncReactor()
	return mempool
}

// NOTE: not thread safe - should only be called once, on startup
func (mem *CListMempool) EnableTxsAvailable() {
	mem.txsAvailable = make(chan struct{}, 1)
}

// SetLogger sets the Logger.
func (mem *CListMempool) SetLogger(l log.Logger) {
	mem.logger = l
}

// WithPreCheck sets a filter for the mempool to reject a tx if f(tx) returns
// false. This is ran before CheckTx. Only applies to the first created block.
// After that, Update overwrites the existing value.
func WithPreCheck(f PreCheckFunc) CListMempoolOption {
	return func(mem *CListMempool) { mem.preCheck = f }
}

// WithPostCheck sets a filter for the mempool to reject a tx if f(tx) returns
// false. This is ran after CheckTx. Only applies to the first created block.
// After that, Update overwrites the existing value.
func WithPostCheck(f PostCheckFunc) CListMempoolOption {
	return func(mem *CListMempool) { mem.postCheck = f }
}

// WithMetrics sets the metrics.
func WithMetrics(metrics *Metrics) CListMempoolOption {
	return func(mem *CListMempool) { mem.metrics = metrics }
}

func (mem *CListMempool) InitWAL() error {
	var (
		walDir  = mem.config.WalDir()
		walFile = walDir + "/wal"
	)

	const perm = 0700
	if err := tmos.EnsureDir(walDir, perm); err != nil {
		return err
	}

	af, err := auto.OpenAutoFile(walFile)
	if err != nil {
		return fmt.Errorf("can't open autofile %s: %w", walFile, err)
	}

	mem.wal = af
	return nil
}

func (mem *CListMempool) CloseWAL() {
	if err := mem.wal.Close(); err != nil {
		mem.logger.Error("Error closing WAL", "err", err)
	}
	mem.wal = nil
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) Lock() {
	mem.updateMtx.Lock()
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) Unlock() {
	mem.updateMtx.Unlock()
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) Size() int {
	return mem.txs.Len()
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) TxsBytes() int64 {
	return atomic.LoadInt64(&mem.txsBytes)
}

// Lock() must be help by the caller during execution.
func (mem *CListMempool) FlushAppConn() error {
	_, err := mem.proxyAppConn.FlushSync()
	return err
}

// XXX: Unsafe! Calling Flush may leave mempool in inconsistent state.
func (mem *CListMempool) Flush() {
	mem.updateMtx.Lock()
	defer mem.updateMtx.Unlock()

	_ = atomic.SwapInt64(&mem.txsBytes, 0)
	mem.cache.Reset()

	for e := mem.txs.Front(); e != nil; e = e.Next() {
		mem.txs.Remove(e)
		e.DetachPrev()
	}

	mem.txsMap.Range(func(key, _ interface{}) bool {
		mem.txsMap.Delete(key)
		return true
	})
}

// TxsFront returns the first transaction in the ordered list for peer
// goroutines to call .NextWait() on.
// FIXME: leaking implementation details!
//
// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) TxsFront() *clist.CElement {
	return mem.txs.Front()
}

// TxsWaitChan returns a channel to wait on transactions. It will be closed
// once the mempool is not empty (ie. the internal `mem.txs` has at least one
// element)
//
// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) TxsWaitChan() <-chan struct{} {
	return mem.txs.WaitChan()
}

// It blocks if we're waiting on Update() or Reap().
// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) CheckTxSync(tx types.Tx, txInfo TxInfo) (res *abci.Response, err error) {
	mem.updateMtx.RLock()
	// use defer to unlock mutex because application (*local client*) might panic
	defer mem.updateMtx.RUnlock()

	if err = mem.prepareCheckTx(tx, txInfo); err != nil {
		return res, err
	}

	// CONTRACT: `app.CheckTxSync()` should check whether `GasWanted` is valid (0 <= GasWanted <= block.masGas)
	var r *abci.ResponseCheckTx
	r, err = mem.proxyAppConn.CheckTxSync(abci.RequestCheckTx{Tx: tx})
	if err != nil {
		return res, err
	}

	res = abci.ToResponseCheckTx(*r)
	mem.reqResCb(tx, txInfo.SenderID, txInfo.SenderP2PID, res, nil)
	return res, err
}

// cb: A callback from the CheckTx command.
//     It gets called from another goroutine.
//
// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) CheckTxAsync(tx types.Tx, txInfo TxInfo, prepareCb func(error),
	checkTxCb func(*abci.Response)) {
	mem.chReqCheckTx <- &requestCheckTxAsync{tx: tx, txInfo: txInfo, prepareCb: prepareCb, checkTxCb: checkTxCb}
}

func (mem *CListMempool) checkTxAsyncReactor() {
	for req := range mem.chReqCheckTx {
		mem.checkTxAsync(req.tx, req.txInfo, req.prepareCb, req.checkTxCb)
	}
}

// It blocks if we're waiting on Update() or Reap().
func (mem *CListMempool) checkTxAsync(tx types.Tx, txInfo TxInfo, prepareCb func(error),
	checkTxCb func(*abci.Response)) {
	mem.updateMtx.RLock()
	defer func() {
		if r := recover(); r != nil {
			mem.updateMtx.RUnlock()
			panic(r)
		}
	}()

	err := mem.prepareCheckTx(tx, txInfo)
	if prepareCb != nil {
		prepareCb(err)
	}
	if err != nil {
		mem.updateMtx.RUnlock()
		return
	}

	// CONTRACT: `app.CheckTxAsync()` should check whether `GasWanted` is valid (0 <= GasWanted <= block.masGas)
	mem.proxyAppConn.CheckTxAsync(abci.RequestCheckTx{Tx: tx}, func(res *abci.Response) {
		mem.reqResCb(tx, txInfo.SenderID, txInfo.SenderP2PID, res, func(response *abci.Response) {
			if checkTxCb != nil {
				checkTxCb(response)
			}
			mem.updateMtx.RUnlock()
		})
	})
}

// CONTRACT: `caller` should held `mem.updateMtx.RLock()`
func (mem *CListMempool) prepareCheckTx(tx types.Tx, txInfo TxInfo) error {
	// For keeping the consistency between `mem.txs` and `mem.txsMap`
	if _, ok := mem.txsMap.Load(TxKey(tx)); ok {
		return ErrTxInMap
	}

	txSize := len(tx)

	if err := mem.isFull(txSize); err != nil {
		return err
	}

	if txSize > mem.config.MaxTxBytes {
		return ErrTxTooLarge{mem.config.MaxTxBytes, txSize}
	}

	if mem.preCheck != nil {
		if err := mem.preCheck(tx); err != nil {
			return ErrPreCheck{err}
		}
	}

	// NOTE: writing to the WAL and calling proxy must be done before adding tx
	// to the cache. otherwise, if either of them fails, next time CheckTx is
	// called with tx, ErrTxInCache will be returned without tx being checked at
	// all even once.
	if mem.wal != nil {
		// TODO: Notify administrators when WAL fails
		_, err := mem.wal.Write(append([]byte(tx), newline...))
		if err != nil {
			return fmt.Errorf("wal.Write: %w", err)
		}
	}

	// NOTE: proxyAppConn may error if tx buffer is full
	if err := mem.proxyAppConn.Error(); err != nil {
		return err
	}

	if !mem.cache.Push(tx) {
		// Record a new sender for a tx we've already seen.
		// Note it's possible a tx is still in the cache but no longer in the mempool
		// (eg. after committing a block, txs are removed from mempool but not cache),
		// so we only record the sender for txs still in the mempool.
		if e, ok := mem.txsMap.Load(TxKey(tx)); ok {
			memTx := e.(*clist.CElement).Value.(*mempoolTx)
			memTx.senders.LoadOrStore(txInfo.SenderID, true)
			// TODO: consider punishing peer for dups,
			// its non-trivial since invalid txs can become valid,
			// but they can spam the same tx with little cost to them atm.
		}

		return ErrTxInCache
	}

	// reserve mempool that should be called just before calling `mem.proxyAppConn.CheckTxAsync()`
	if err := mem.reserve(int64(txSize)); err != nil {
		// remove from cache
		mem.cache.Remove(tx)
		return err
	}

	return nil
}

// Global callback that will be called after every ABCI response.
// Having a single global callback avoids needing to set a callback for each request.
// However, processing the checkTx response requires the peerID (so we can track which txs we heard from who),
// and peerID is not included in the ABCI request, so we have to set request-specific callbacks that
// include this information. If we're not in the midst of a recheck, this function will just return,
// so the request specific callback can do the work.
//
// When rechecking, we don't need the peerID, so the recheck callback happens
// here.
func (mem *CListMempool) globalCb(req *abci.Request, res *abci.Response) {
	checkTxReq := req.GetCheckTx()
	if checkTxReq == nil {
		return
	}

	if checkTxReq.Type == abci.CheckTxType_Recheck {
		mem.metrics.RecheckCount.Add(1)
		mem.resCbRecheck(req, res)

		// update metrics
		mem.metrics.Size.Set(float64(mem.Size()))
	}
}

// Request specific callback that should be set on individual reqRes objects
// to incorporate local information when processing the response.
// This allows us to track the peer that sent us this tx, so we can avoid sending it back to them.
// NOTE: alternatively, we could include this information in the ABCI request itself.
//
// External callers of CheckTx, like the RPC, can also pass an externalCb through here that is called
// when all other response processing is complete.
//
// Used in CheckTx to record PeerID who sent us the tx.
func (mem *CListMempool) reqResCb(
	tx []byte,
	peerID uint16,
	peerP2PID p2p.ID,
	res *abci.Response,
	externalCb func(*abci.Response),
) {
	mem.resCbFirstTime(tx, peerID, peerP2PID, res)

	// update metrics
	mem.metrics.Size.Set(float64(mem.Size()))

	// passed in by the caller of CheckTx, eg. the RPC
	if externalCb != nil {
		externalCb(res)
	}
}

// Called from:
//  - resCbFirstTime (lock not held) if tx is valid
func (mem *CListMempool) addTx(memTx *mempoolTx) {
	e := mem.txs.PushBack(memTx)
	mem.txsMap.Store(TxKey(memTx.tx), e)
	atomic.AddInt64(&mem.txsBytes, int64(len(memTx.tx)))
	mem.metrics.TxSizeBytes.Observe(float64(len(memTx.tx)))
}

// Called from:
//  - Update (lock held) if tx was committed
// 	- resCbRecheck (lock not held) if tx was invalidated
func (mem *CListMempool) removeTx(tx types.Tx, elem *clist.CElement, removeFromCache bool) {
	mem.txs.Remove(elem)
	elem.DetachPrev()
	mem.txsMap.Delete(TxKey(tx))
	atomic.AddInt64(&mem.txsBytes, int64(-len(tx)))

	if removeFromCache {
		mem.cache.Remove(tx)
	}
}

// RemoveTxByKey removes a transaction from the mempool by its TxKey index.
func (mem *CListMempool) RemoveTxByKey(txKey [TxKeySize]byte, removeFromCache bool) {
	if e, ok := mem.txsMap.Load(txKey); ok {
		memTx := e.(*clist.CElement).Value.(*mempoolTx)
		if memTx != nil {
			mem.removeTx(memTx.tx, e.(*clist.CElement), removeFromCache)
		}
	}
}

func (mem *CListMempool) isFull(txSize int) error {
	var (
		memSize  = mem.Size()
		txsBytes = mem.TxsBytes()
	)

	if memSize >= mem.config.Size || int64(txSize)+txsBytes > mem.config.MaxTxsBytes {
		return ErrMempoolIsFull{
			memSize, mem.config.Size,
			txsBytes, mem.config.MaxTxsBytes,
		}
	}

	return nil
}

func (mem *CListMempool) reserve(txSize int64) error {
	mem.reservedMtx.Lock()
	defer mem.reservedMtx.Unlock()

	var (
		memSize  = mem.Size()
		txsBytes = mem.TxsBytes()
	)

	if memSize+mem.reserved >= mem.config.Size || txSize+mem.reservedBytes+txsBytes > mem.config.MaxTxsBytes {
		return ErrMempoolIsFull{
			memSize + mem.reserved, mem.config.Size,
			txsBytes + mem.reservedBytes, mem.config.MaxTxsBytes,
		}
	}

	mem.reserved++
	mem.reservedBytes += txSize
	return nil
}

func (mem *CListMempool) releaseReserve(txSize int64) {
	mem.reservedMtx.Lock()
	defer mem.reservedMtx.Unlock()

	mem.reserved--
	mem.reservedBytes -= txSize
}

// callback, which is called after the app checked the tx for the first time.
//
// The case where the app checks the tx for the second and subsequent times is
// handled by the resCbRecheck callback.
func (mem *CListMempool) resCbFirstTime(
	tx []byte,
	peerID uint16,
	peerP2PID p2p.ID,
	res *abci.Response,
) {
	switch r := res.Value.(type) {
	case *abci.Response_CheckTx:
		if r.CheckTx.Code == abci.CodeTypeOK {
			memTx := &mempoolTx{
				height:    mem.height,
				gasWanted: r.CheckTx.GasWanted,
				tx:        tx,
			}
			memTx.senders.Store(peerID, true)
			mem.addTx(memTx)
			mem.logger.Debug("added good transaction",
				"tx", txID(tx),
				"res", r,
				"height", memTx.height,
				"total", mem.Size(),
			)
			mem.notifyTxsAvailable()
		} else {
			// ignore bad transaction
			mem.logger.Debug("rejected bad transaction",
				"tx", txID(tx), "peerID", peerP2PID, "res", r)
			mem.metrics.FailedTxs.Add(1)
			if !mem.config.KeepInvalidTxsInCache {
				// remove from cache (it might be good later)
				mem.cache.Remove(tx)
			}
		}

		// release `reserve` regardless it's OK or not (it might be good later)
		mem.releaseReserve(int64(len(tx)))
	default:
		// ignore other messages
	}
}

// callback, which is called after the app rechecked the tx.
//
// The case where the app checks the tx for the first time is handled by the
// resCbFirstTime callback.
func (mem *CListMempool) resCbRecheck(req *abci.Request, res *abci.Response) {
	switch r := res.Value.(type) {
	case *abci.Response_CheckTx:
		tx := req.GetCheckTx().Tx
		txHash := TxKey(tx)
		e, ok := mem.txsMap.Load(txHash)
		if !ok {
			mem.logger.Debug("re-CheckTx transaction does not exist", "expected", types.Tx(tx))
			return
		}
		var postCheckErr error
		if r.CheckTx.Code == abci.CodeTypeOK {
			if mem.postCheck == nil {
				return
			}
			postCheckErr = mem.postCheck(tx, r.CheckTx)
			if postCheckErr == nil {
				return
			}
			r.CheckTx.MempoolError = postCheckErr.Error()
		}
		celem := e.(*clist.CElement)
		// Tx became invalidated due to newly committed block.
		mem.logger.Debug("tx is no longer valid", "tx", txID(tx), "res", r, "err", postCheckErr)
		// NOTE: we remove tx from the cache because it might be good later
		mem.removeTx(tx, celem, !mem.config.KeepInvalidTxsInCache)
	default:
		// ignore other messages
	}
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) TxsAvailable() <-chan struct{} {
	return mem.txsAvailable
}

func (mem *CListMempool) notifyTxsAvailable() {
	if mem.Size() == 0 {
		mem.logger.Info("notified txs available but mempool is empty!")
	}
	if mem.txsAvailable != nil && !mem.notifiedTxsAvailable {
		// channel cap is 1, so this will send once
		mem.notifiedTxsAvailable = true
		select {
		case mem.txsAvailable <- struct{}{}:
		default:
		}
	}
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) ReapMaxBytesMaxGas(maxBytes, maxGas int64) types.Txs {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	var totalGas int64

	// TODO: we will get a performance boost if we have a good estimate of avg
	// size per tx, and set the initial capacity based off of that.
	// txs := make([]types.Tx, 0, tmmath.MinInt(mem.txs.Len(), max/mem.avgTxSize))
	txs := make([]types.Tx, 0, mem.txs.Len())
	protoTxs := tmproto.Data{}
	for e := mem.txs.Front(); e != nil; e = e.Next() {
		memTx := e.Value.(*mempoolTx)

		protoTxs.Txs = append(protoTxs.Txs, memTx.tx)
		// Check total size requirement
		if maxBytes > -1 && int64(protoTxs.Size()) > maxBytes {
			return txs
		}
		// Check total gas requirement.
		// If maxGas is negative, skip this check.
		// Since newTotalGas < masGas, which
		// must be non-negative, it follows that this won't overflow.
		newTotalGas := totalGas + memTx.gasWanted
		if maxGas > -1 && newTotalGas > maxGas {
			return txs
		}
		totalGas = newTotalGas
		txs = append(txs, memTx.tx)
	}
	return txs
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) ReapMaxBytesMaxGasMaxTxs(maxBytes, maxGas, maxTxs int64) types.Txs {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	var totalGas int64

	if maxTxs <= 0 {
		maxTxs = int64(mem.txs.Len())
	}

	// TODO: we will get a performance boost if we have a good estimate of avg
	// size per tx, and set the initial capacity based off of that.
	// txs := make([]types.Tx, 0, tmmath.MinInt(mem.txs.Len(), max/mem.avgTxSize))
	txs := make([]types.Tx, 0, mem.txs.Len())
	protoTxs := tmproto.Data{}
	for e := mem.txs.Front(); e != nil && len(txs) < int(maxTxs); e = e.Next() {
		memTx := e.Value.(*mempoolTx)

		protoTxs.Txs = append(protoTxs.Txs, memTx.tx)
		// Check total size requirement
		if maxBytes > -1 && int64(protoTxs.Size()) > maxBytes {
			return txs
		}
		// Check total gas requirement.
		// If maxGas is negative, skip this check.
		// Since newTotalGas < masGas, which
		// must be non-negative, it follows that this won't overflow.
		newTotalGas := totalGas + memTx.gasWanted
		if maxGas > -1 && newTotalGas > maxGas {
			return txs
		}
		totalGas = newTotalGas
		txs = append(txs, memTx.tx)
	}
	return txs
}

// Safe for concurrent use by multiple goroutines.
func (mem *CListMempool) ReapMaxTxs(max int) types.Txs {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	if max < 0 {
		max = mem.txs.Len()
	}

	txs := make([]types.Tx, 0, tmmath.MinInt(mem.txs.Len(), max))
	for e := mem.txs.Front(); e != nil && len(txs) <= max; e = e.Next() {
		memTx := e.Value.(*mempoolTx)
		txs = append(txs, memTx.tx)
	}
	return txs
}

// Lock() must be held by the caller during execution.
func (mem *CListMempool) Update(
	block *types.Block,
	deliverTxResponses []*abci.ResponseDeliverTx,
	preCheck PreCheckFunc,
	postCheck PostCheckFunc,
) (err error) {
	// Set height
	mem.height = block.Height
	mem.notifiedTxsAvailable = false

	if preCheck != nil {
		mem.preCheck = preCheck
	}
	if postCheck != nil {
		mem.postCheck = postCheck
	}

	for i, tx := range block.Txs {
		if deliverTxResponses[i].Code == abci.CodeTypeOK {
			// Add valid committed tx to the cache (if missing).
			_ = mem.cache.Push(tx)
		} else if !mem.config.KeepInvalidTxsInCache {
			// Allow invalid transactions to be resubmitted.
			mem.cache.Remove(tx)
		}

		// Remove committed tx from the mempool.
		//
		// Note an evil proposer can drop valid txs!
		// Mempool before:
		//   100 -> 101 -> 102
		// Block, proposed by an evil proposer:
		//   101 -> 102
		// Mempool after:
		//   100
		// https://github.com/tendermint/tendermint/issues/3322.
		if e, ok := mem.txsMap.Load(TxKey(tx)); ok {
			mem.removeTx(tx, e.(*clist.CElement), false)
		}
	}

	if mem.config.Recheck {
		// recheck non-committed txs to see if they became invalid
		recheckStartTime := time.Now().UnixNano()

		_, err = mem.proxyAppConn.BeginRecheckTxSync(abci.RequestBeginRecheckTx{
			Header: types.OC2PB.Header(&block.Header),
		})
		if err != nil {
			mem.logger.Error("error in proxyAppConn.BeginRecheckTxSync", "err", err)
		}
		mem.logger.Debug("recheck txs", "numtxs", mem.Size(), "height", block.Height)
		mem.recheckTxs()
		_, err = mem.proxyAppConn.EndRecheckTxSync(abci.RequestEndRecheckTx{Height: block.Height})
		if err != nil {
			mem.logger.Error("error in proxyAppConn.EndRecheckTxSync", "err", err)
		}

		recheckEndTime := time.Now().UnixNano()

		recheckTimeMs := float64(recheckEndTime-recheckStartTime) / 1000000
		mem.metrics.RecheckTime.Set(recheckTimeMs)
	}

	// notify there're some txs left.
	if mem.Size() > 0 {
		mem.notifyTxsAvailable()
	}

	// Update metrics
	mem.metrics.Size.Set(float64(mem.Size()))

	return err
}

func (mem *CListMempool) recheckTxs() {
	if mem.Size() == 0 {
		return
	}

	wg := sync.WaitGroup{}

	// Push txs to proxyAppConn
	// NOTE: globalCb may be called concurrently.
	for e := mem.txs.Front(); e != nil; e = e.Next() {
		wg.Add(1)

		memTx := e.Value.(*mempoolTx)
		req := abci.RequestCheckTx{
			Tx:   memTx.tx,
			Type: abci.CheckTxType_Recheck,
		}

		mem.proxyAppConn.CheckTxAsync(req, func(res *abci.Response) {
			wg.Done()
		})
	}

	mem.proxyAppConn.FlushAsync(func(res *abci.Response) {})
	wg.Wait()
}

//--------------------------------------------------------------------------------

// mempoolTx is a transaction that successfully ran
type mempoolTx struct {
	height    int64    // height that this tx had been validated in
	gasWanted int64    // amount of gas this tx states it will require
	tx        types.Tx //

	// ids of peers who've sent us this tx (as a map for quick lookups).
	// senders: PeerID -> bool
	senders sync.Map
}

// Height returns the height for this transaction
func (memTx *mempoolTx) Height() int64 {
	return atomic.LoadInt64(&memTx.height)
}

//--------------------------------------------------------------------------------

type txCache interface {
	Reset()
	Push(tx types.Tx) bool
	Remove(tx types.Tx)
}

// mapTxCache maintains a LRU cache of transactions. This only stores the hash
// of the tx, due to memory concerns.
type mapTxCache struct {
	mtx      tmsync.Mutex
	size     int
	cacheMap map[[TxKeySize]byte]*list.Element
	list     *list.List
}

var _ txCache = (*mapTxCache)(nil)

// newMapTxCache returns a new mapTxCache.
func newMapTxCache(cacheSize int) *mapTxCache {
	return &mapTxCache{
		size:     cacheSize,
		cacheMap: make(map[[TxKeySize]byte]*list.Element, cacheSize),
		list:     list.New(),
	}
}

// Reset resets the cache to an empty state.
func (cache *mapTxCache) Reset() {
	cache.mtx.Lock()
	cache.cacheMap = make(map[[TxKeySize]byte]*list.Element, cache.size)
	cache.list.Init()
	cache.mtx.Unlock()
}

// Push adds the given tx to the cache and returns true. It returns
// false if tx is already in the cache.
func (cache *mapTxCache) Push(tx types.Tx) bool {
	cache.mtx.Lock()
	defer cache.mtx.Unlock()

	// Use the tx hash in the cache
	txHash := TxKey(tx)
	if moved, exists := cache.cacheMap[txHash]; exists {
		cache.list.MoveToBack(moved)
		return false
	}

	if cache.list.Len() >= cache.size {
		popped := cache.list.Front()
		if popped != nil {
			poppedTxHash := popped.Value.([TxKeySize]byte)
			delete(cache.cacheMap, poppedTxHash)
			cache.list.Remove(popped)
		}
	}
	e := cache.list.PushBack(txHash)
	cache.cacheMap[txHash] = e
	return true
}

// Remove removes the given tx from the cache.
func (cache *mapTxCache) Remove(tx types.Tx) {
	cache.mtx.Lock()
	txHash := TxKey(tx)
	popped := cache.cacheMap[txHash]
	delete(cache.cacheMap, txHash)
	if popped != nil {
		cache.list.Remove(popped)
	}

	cache.mtx.Unlock()
}

type nopTxCache struct{}

var _ txCache = (*nopTxCache)(nil)

func (nopTxCache) Reset()             {}
func (nopTxCache) Push(types.Tx) bool { return true }
func (nopTxCache) Remove(types.Tx)    {}

//--------------------------------------------------------------------------------

// TxKey is the fixed length array hash used as the key in maps.
func TxKey(tx types.Tx) [TxKeySize]byte {
	return sha256.Sum256(tx)
}

// txID is a hash of the Tx.
func txID(tx []byte) []byte {
	return types.Tx(tx).Hash()
}
