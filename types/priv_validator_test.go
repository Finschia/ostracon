package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPvKeyTypeByAddress(t *testing.T) {
	for i := 0; i < 1000; i++ {
		keyType := randomKeyType()
		pv := NewMockPV(keyType)
		pubKey, _ := pv.GetPubKey()
		assert.True(t, keyType == PvKeyTypeByPubKey(pubKey))
	}
}
