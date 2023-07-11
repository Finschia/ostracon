package mempool

import (
	"crypto/rand"
	"testing"

	"github.com/Finschia/ostracon/types"

	"github.com/stretchr/testify/require"
)

func TestCacheBasic(t *testing.T) {
	size := 10
	cache := NewLRUTxCache(size)
	require.Equal(t, 0, cache.GetList().Len())
	for i := 0; i < size; i++ {
		cache.Push(types.Tx{byte(i)})
	}
	require.Equal(t, size, cache.GetList().Len())
	cache.Push(types.Tx{byte(0)}) // MoveToBack
	require.Equal(t, size, cache.GetList().Len())
	cache.Push(types.Tx{byte(size + 1)}) // Cache out
	require.Equal(t, size, cache.GetList().Len())
	cache.Reset()
	require.Equal(t, 0, cache.GetList().Len())
}

func TestCacheRemove(t *testing.T) {
	cache := NewLRUTxCache(100)
	numTxs := 10

	txs := make([][]byte, numTxs)
	for i := 0; i < numTxs; i++ {
		// probability of collision is 2**-256
		txBytes := make([]byte, 32)
		_, err := rand.Read(txBytes)
		require.NoError(t, err)

		txs[i] = txBytes
		cache.Push(txBytes)

		// make sure its added to both the linked list and the map
		require.Equal(t, i+1, len(cache.cacheMap))
		require.Equal(t, i+1, cache.list.Len())
	}

	for i := 0; i < numTxs; i++ {
		cache.Remove(txs[i])
		// make sure its removed from both the map and the linked list
		require.Equal(t, numTxs-(i+1), len(cache.cacheMap))
		require.Equal(t, numTxs-(i+1), cache.list.Len())
	}
}
