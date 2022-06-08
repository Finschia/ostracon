package privval

import (
	"testing"

	"github.com/line/ostracon/crypto"
)

func TestWithMockKMS(t *testing.T) {
	dir := t.TempDir()
	WithMockKMS(t, dir, "test", func(addr string, privKey crypto.PrivKey) {})
}
