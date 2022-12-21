package log

import (
	tmbytes "github.com/line/ostracon/libs/bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewLazySprintf(t *testing.T) {
	format := "echo:%s"
	args := make([]interface{}, 0, 1)
	args = append(args, "hello")
	expected := LazySprintf{format: format, args: args}
	actual := NewLazySprintf(format, args...)
	require.Equal(t, expected.String(), actual.String())
}

func TestNewLazyBlockHash(t *testing.T) {
	block := testHashable{}
	expected := LazyBlockHash{block: block}
	actual := NewLazyBlockHash(block)
	require.Equal(t, expected.String(), actual.String())
}

type testHashable struct{}

func (testHashable) Hash() tmbytes.HexBytes {
	return []byte{0}
}
