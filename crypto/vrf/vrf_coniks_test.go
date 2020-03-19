package vrf

import (
	"bytes"
	"testing"

	coniksimpl "github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestKeyPairCompatibilityConiks(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	publicKey, _ := privateKey.PubKey().(ed25519.PubKeyEd25519)

	privateKey2 := coniksimpl.PrivateKey(make([]byte, 64))
	copy(privateKey2, privateKey[:])
	publicKey2, _ := privateKey2.Public()
	if bytes.Compare(publicKey[:], publicKey2) != 0 {
		t.Error("public key is not matched(coniks key -> tm key")
	}

	privateKey2, _ = coniksimpl.GenerateKey(nil)
	publicKey2, _ = privateKey2.Public()

	privateKey = ed25519.PrivKeyEd25519{}
	copy(privateKey[:], privateKey2)
	publicKey = privateKey.PubKey().(ed25519.PubKeyEd25519)
	if bytes.Compare(publicKey[:], publicKey2) != 0 {
		t.Error("public key is not matched(tm key -> coniks key")
	}
}
