package vrf

import (
	"bytes"
	"testing"

	coniksimpl "github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestProveAndVerifyConiks(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	publicKey, _ := privateKey.PubKey().(ed25519.PubKeyEd25519)

	t.Logf("private key: [%s]", enc(privateKey[:]))
	t.Logf("public  key: [%s]", enc(publicKey[:]))

	vrfImpl := newVrfEd25519Coniks()
	message := []byte("hello, world")
	proof, err1 := vrfImpl.Prove(privateKey, message)
	if err1 != nil {
		t.Fatalf("failed to prove: %s", err1)
	}
	t.Logf("proof: %s", enc(proof[:]))

	hash1, err2 := vrfImpl.ProofToHash(proof)
	if err2 != nil {
		t.Fatalf("failed to hash: %s", err2)
	}
	t.Logf("hash for \"%s\": %s", message, hash1.ToInt())

	verified, err3 := vrfImpl.Verify(publicKey, proof, message)
	if err3 != nil {
		t.Errorf("failed to verify: %s", err3)
	} else if !verified {
		t.Errorf("incompatible output")
	}
}

func TestKeyPairCompatibilityConiks(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	publicKey, _ := privateKey.PubKey().(ed25519.PubKeyEd25519)

	privateKey2 := coniksimpl.PrivateKey(make([]byte, 64))
	copy(privateKey2, privateKey[:])
	publicKey2, _ := privateKey2.Public()
	if !bytes.Equal(publicKey[:], publicKey2) {
		t.Error("public key is not matched(coniks key -> tm key")
	}

	privateKey2, _ = coniksimpl.GenerateKey(nil)
	publicKey2, _ = privateKey2.Public()

	privateKey = ed25519.PrivKeyEd25519{}
	copy(privateKey[:], privateKey2)
	publicKey = privateKey.PubKey().(ed25519.PubKeyEd25519)
	if !bytes.Equal(publicKey[:], publicKey2) {
		t.Error("public key is not matched(tm key -> coniks key")
	}
}
