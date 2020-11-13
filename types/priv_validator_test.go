package types

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/tendermint/libs/rand"
)

func forAllPrivKeyTypes(t *testing.T, exec func(t *testing.T, name string, keyType PrivKeyType)) {
	keyNameAndTypes := []struct {
		name    string
		keyType PrivKeyType
	}{
		{name: "ed25512", keyType: PrivKeyEd25519},
		{name: "composite", keyType: PrivKeyComposite},
		{name: "bls", keyType: PrivKeyBLS}}
	for _, knt := range keyNameAndTypes {
		t.Run(knt.name, func(t *testing.T) {
			exec(t, knt.name, knt.keyType)
		})
	}
}

func randomKeyType() PrivKeyType {
	r := rand.Uint32() % 2
	switch r {
	case 0:
		return PrivKeyEd25519
	case 1:
		return PrivKeyComposite
	}
	return PrivKeyEd25519
}

func TestPvKeyTypeByAddress(t *testing.T) {
	for i := 0; i < 1000; i++ {
		keyType := randomKeyType()
		pv := NewMockPV(keyType)
		pubKey, _ := pv.GetPubKey()
		assert.True(t, keyType == PrivKeyTypeByPubKey(pubKey))
	}
}
