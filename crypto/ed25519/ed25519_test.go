package ed25519_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Finschia/ostracon/crypto"
	"github.com/Finschia/ostracon/crypto/ed25519"
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

	// *** If the combination of (pubkey, message, proof) is incorrect ***
	// invalid message
	inValidMessage, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
	_, err1 := pubKey.VRFVerify(proof, inValidMessage)
	assert.Error(t, err1)

	// invalid pubkey
	invalidPrivKey := ed25519.GenPrivKey()
	invalidPubkey := invalidPrivKey.PubKey()
	_, err2 := invalidPubkey.VRFVerify(proof, message)
	assert.Error(t, err2)

	// invalid proof
	invalidProof, _ := invalidPrivKey.VRFProve(message)
	_, err3 := pubKey.VRFVerify(invalidProof, message)
	assert.Error(t, err3)
}
