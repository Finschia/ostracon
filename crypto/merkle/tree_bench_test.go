package merkle

import (
	"testing"

	"github.com/line/ostracon/crypto/tmhash"
	tmrand "github.com/line/ostracon/libs/rand"
)

func BenchmarkHashFromByteSlices(b *testing.B) {
	const total = 4000
	slices := make([][]byte, total)
	for j := 0; j < total; j++ {
		slices[j] = tmrand.Bytes(tmhash.Size)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashFromByteSlices(slices)
	}
}

func BenchmarkHashFromByteSlicesParallel(b *testing.B) {
	const total = 4000
	slices := make([][]byte, total)
	for j := 0; j < total; j++ {
		slices[j] = tmrand.Bytes(tmhash.Size)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashFromByteSlicesParallel(slices, 0)
	}
}
