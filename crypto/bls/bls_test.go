package bls_test

import (
	"bytes"
	"fmt"
	"testing"

	b "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/go-amino"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestPrivKeyBLS12_Bytes(t *testing.T) {
	pk := bls.GenPrivKey()
	b := pk.Bytes()
	if len(b) != 37 {
		t.Fatalf("bytes length: %d != %d", len(b), 37)
	}
	if !bytes.Equal(b, pk.Bytes()) {
		t.Errorf("bytes is not identical by call")
	}
}

func TestPrivKeyBLS12_Sign(t *testing.T) {
	msg := []byte{0, 1, 2, 3, 4, 5}
	sign, err := bls.GenPrivKey().Sign(msg)
	if err != nil {
		t.Fatalf("Signing failed: %s", err)
	}
	if sign == nil {
		t.Errorf("Signature is nil")
	}
	if len(sign) != bls.SignatureSize {
		t.Errorf("Unexpected signature size: %d != %d", len(sign), bls.SignatureSize)
	}
}

func TestPrivKeyBLS12_PubKey(t *testing.T) {
	pubKey := bls.GenPrivKey().PubKey()
	if pubKey == nil {
		t.Errorf("Public key is nil")
	}
}

func TestPrivKeyBLS12_Equals(t *testing.T) {
	privKey := bls.GenPrivKey()
	anotherPrivKey := bls.GenPrivKey()
	if !privKey.Equals(privKey) {
		t.Error("A is not identical")
	}
	if privKey.Equals(anotherPrivKey) || anotherPrivKey.Equals(privKey) {
		t.Error("Different A and B were determined to be identical")
	}

	cdc := amino.NewCodec()
	json, err := cdc.MarshalJSON(privKey)
	if err != nil {
		t.Fatalf("Marshalling failed: %s", err)
	}
	var restoredPrivKey = bls.PrivKeyBLS12{}
	err = cdc.UnmarshalJSON(json, &restoredPrivKey)
	if err != nil {
		t.Fatalf("Unmarshalling failed: %s", err)
	}
	if !privKey.Equals(restoredPrivKey) || !restoredPrivKey.Equals(privKey) {
		t.Errorf("Restored A was not identical")
	}

	ed25519PrivKey := ed25519.GenPrivKey()
	if privKey.Equals(ed25519PrivKey) {
		t.Errorf("Different types of keys were matched")
	}
}

func TestPubKeyBLS12_Address(t *testing.T) {
	privKey := bls.GenPrivKey()
	pubKey := privKey.PubKey()
	address := pubKey.Address()
	if len(address) != 20 {
		t.Errorf("Address length %d is not %d", len(address), 20)
	}
}

func TestPubKeyBLS12_Bytes(t *testing.T) {
	privKey := bls.GenPrivKey()
	pubKey := privKey.PubKey()
	b := pubKey.Bytes()
	if len(b) != 53 {
		t.Errorf("Byte length %d is not %d", len(b), 53)
	}
	if !bytes.Equal(b, pubKey.Bytes()) {
		t.Errorf("Difference bytes was generated from the same public key")
	}
	if bytes.Equal(b, bls.GenPrivKey().PubKey().Bytes()) {
		t.Errorf("The same bytes was generated from different public key")
	}
}

func TestPubKeyBLS12_VerifyBytes(t *testing.T) {
	privKey := bls.GenPrivKey()
	pubKey := privKey.PubKey()
	msg := []byte{0, 1, 2, 3, 4, 5}
	sign, _ := privKey.Sign(msg)
	if !pubKey.VerifyBytes(msg, sign) {
		t.Errorf("Signature validation failed for the same message")
	}

	corruptedMessage := make([]byte, len(msg))
	copy(corruptedMessage, msg)
	corruptedMessage[0] ^= 1
	if pubKey.VerifyBytes(corruptedMessage, sign) {
		t.Errorf("Signature validation succeeded for the different messages")
	}

	otherPubKey := bls.GenPrivKey().PubKey()
	if otherPubKey.VerifyBytes(msg, sign) {
		t.Errorf("Signature validation succeeded for the different public key")
	}

	ed25519Sign, _ := ed25519.GenPrivKey().Sign(msg)
	if pubKey.VerifyBytes(msg, ed25519Sign) {
		t.Errorf("Verification accepted by ed25519 signature")
	}

	emptySign := make([]byte, 0)
	if pubKey.VerifyBytes(msg, emptySign) {
		t.Errorf("Verification accepted by empty bytes")
	}

	zeroSign := make([]byte, bls.SignatureSize)
	if pubKey.VerifyBytes(msg, zeroSign) {
		t.Errorf("Verification accepted by zero-filled bytes")
	}
}

func TestPubKeyBLS12_String(t *testing.T) {
	fmt.Printf("%s\n", bls.GenPrivKey().PubKey())
}

func TestPubKeyBLS12_Equals(t *testing.T) {
	privKey := bls.GenPrivKey()
	pubKey := privKey.PubKey()
	samePubKey := privKey.PubKey()
	if !pubKey.Equals(samePubKey) {
		t.Errorf("Different public keys are generated from the same private key")
	}

	anotherPubKey := bls.GenPrivKey().PubKey()
	if pubKey.Equals(anotherPubKey) {
		t.Errorf("The same public keys are generated from different private keys")
	}

	ed25519PubKey := ed25519.GenPrivKey().PubKey()
	if pubKey.Equals(ed25519PubKey) {
		t.Errorf("Got a match on a different kind of key")
	}
}

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
