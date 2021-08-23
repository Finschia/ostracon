package ed25519_test

import (
	"encoding/hex"
	coniks "github.com/coniks-sys/coniks-go/crypto/vrf"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/ed25519"
)

func TestSignAndValidateEd25519(t *testing.T) {

	privKey := ed25519.GenPrivKey()
	pubKey := privKey.PubKey()

	msg := crypto.CRandBytes(128)
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)

	// Test the signature
	assert.True(t, pubKey.VerifySignature(msg, sig))

	// Mutate the signature, just one bit.
	// TODO: Replace this with a much better fuzzer, tendermint/ed25519/issues/10
	sig[7] ^= byte(0x01)

	assert.False(t, pubKey.VerifySignature(msg, sig))
}

func TestVRFProveAndVRFVerify(t *testing.T) {

	privKey := ed25519.GenPrivKey()
	pubKey := privKey.PubKey()

	message, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	proof, err := privKey.VRFProve(message)
	assert.Nil(t, err)
	assert.NotNil(t, proof)

	output, err := pubKey.VRFVerify(proof, message)
	assert.Nil(t, err)
	assert.NotNil(t, output)

	// error
	{
		message, _ = hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
		output, err = pubKey.VRFVerify(proof, message)
		assert.NotNil(t, err)
		assert.Nil(t, output)
	}

	// invalid
	{
		privateKey, _ := coniks.GenerateKey(nil)
		copy(privKey[:], privateKey)
		pubKey = privKey.PubKey()

		proof, err = privKey.VRFProve(message)
		assert.Nil(t, err)
		assert.NotNil(t, proof)

		output, err = pubKey.VRFVerify(proof, message)
		assert.NotNil(t, err)
		assert.Nil(t, output)
	}
}
