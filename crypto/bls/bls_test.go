package bls_test

import (
	"bytes"
	"fmt"
	"testing"

	b "github.com/herumi/bls-eth-go-binary/bls"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/bls"
)

func TestBasicSignatureFunctions(t *testing.T) {
	privateKey := b.SecretKey{}
	privateKey.SetByCSPRNG()
	publicKey := privateKey.GetPublicKey()

	duplicatedPrivateKey := b.SecretKey{}
	err := duplicatedPrivateKey.Deserialize(privateKey.Serialize())
	if err != nil {
		t.Fatalf("Private key deserialization failed.")
	}

	if len(privateKey.Serialize()) != bls.PrivKeyBLS12Size {
		t.Fatalf("The constant size %d of the private key is different from the actual size %d.",
			bls.PrivKeyBLS12Size, len(privateKey.Serialize()))
	}

	duplicatedPublicKey := b.PublicKey{}
	err = duplicatedPublicKey.Deserialize(publicKey.Serialize())
	if err != nil {
		t.Fatalf("Public key deserialization failed.")
	}

	if len(publicKey.Serialize()) != bls.PubKeyBLS12Size {
		t.Fatalf("The constant size %d of the public key is different from the actual size %d.",
			bls.PubKeyBLS12Size, len(publicKey.Serialize()))
	}

	duplicatedSignature := func(sig *b.Sign) *b.Sign {
		duplicatedSign := b.Sign{}
		err := duplicatedSign.Deserialize(sig.Serialize())
		if err != nil {
			t.Fatalf("Signature deserialization failed.")
		}

		if len(sig.Serialize()) != bls.SignatureSize {
			t.Fatalf("The constant size %d of the signature is different from the actual size %d.",
				bls.SignatureSize, len(sig.Serialize()))
		}
		return &duplicatedSign
	}

	msg := []byte("hello, world")
	for _, privKey := range []b.SecretKey{privateKey, duplicatedPrivateKey} {
		for _, pubKey := range []*b.PublicKey{publicKey, &duplicatedPublicKey} {
			signature := privKey.SignByte(msg)
			if !signature.VerifyByte(pubKey, msg) {
				t.Errorf("Signature verification failed.")
			}

			if !duplicatedSignature(signature).VerifyByte(pubKey, msg) {
				t.Errorf("Signature verification failed.")
			}

			for i := 0; i < len(msg); i++ {
				for j := 0; j < 8; j++ {
					garbled := make([]byte, len(msg))
					copy(garbled, msg)
					garbled[i] ^= 1 << (8 - j - 1)
					if bytes.Equal(msg, garbled) {
						t.Fatalf("Not a barbled message")
					}
					if signature.VerifyByte(pubKey, garbled) {
						t.Errorf("Signature verification was successful against a garbled byte sequence.")
					}
					if duplicatedSignature(signature).VerifyByte(pubKey, garbled) {
						t.Errorf("Signature verification was successful against a garbled byte sequence.")
					}
				}
			}
		}
	}
}

func TestSignatureAggregation(t *testing.T) {
	publicKeys := make([]b.PublicKey, 25)
	aggregatedSignature := b.Sign{}
	aggregatedPublicKey := b.PublicKey{}
	msg := []byte("hello, world")
	for i := 0; i < len(publicKeys); i++ {
		privateKey := b.SecretKey{}
		privateKey.SetByCSPRNG()
		publicKeys[i] = *privateKey.GetPublicKey()
		aggregatedSignature.Add(privateKey.SignByte(msg))
		aggregatedPublicKey.Add(&publicKeys[i])
	}

	if !aggregatedSignature.FastAggregateVerify(publicKeys, msg) {
		t.Errorf("Aggregated signature verification failed.")
	}
	if !aggregatedSignature.VerifyByte(&aggregatedPublicKey, msg) {
		t.Errorf("Aggregated signature verification failed.")
	}
}

func TestSignAndValidateBLS12(t *testing.T) {
	privKey := bls.GenPrivKey()
	pubKey := privKey.PubKey()

	msg := crypto.CRandBytes(128)
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)
	fmt.Printf("restoring signature: %x\n", sig)

	// Test the signature
	assert.True(t, pubKey.VerifyBytes(msg, sig))

	// Mutate the signature, just one bit.
	// TODO: Replace this with a much better fuzzer, tendermint/ed25519/issues/10
	sig[7] ^= byte(0x01)

	assert.False(t, pubKey.VerifyBytes(msg, sig))
}
