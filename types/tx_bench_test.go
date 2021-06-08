package types

import (
	"testing"
)

func BenchmarkTx_Hash(b *testing.B) {
	const (
		total = 4000
		size  = 300
	)
	txs := makeTxs(total, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txs.Hash()
	}
}
