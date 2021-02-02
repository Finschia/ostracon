package sr25519_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	cryptoamino "github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/crypto/sr25519"
)

func TestSignAndValidateSr25519(t *testing.T) {

	privKey := sr25519.GenPrivKey()
	pubKey := privKey.PubKey()

	msg := crypto.CRandBytes(128)
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)

	// Test the signature
	assert.True(t, pubKey.VerifyBytes(msg, sig))
	assert.True(t, pubKey.VerifyBytes(msg, sig))

	// Mutate the signature, just one bit.
	// TODO: Replace this with a much better fuzzer, tendermint/ed25519/issues/10
	sig[7] ^= byte(0x01)

	assert.False(t, pubKey.VerifyBytes(msg, sig))
}

func TestMarshal(t *testing.T) {
	cdc := amino.NewCodec()
	cryptoamino.RegisterAmino(cdc)

	privKey := sr25519.GenPrivKey()
	pubKey := privKey.PubKey()

	require.Equal(t, cdc.MustMarshalBinaryBare(privKey), privKey.Bytes())
	require.Equal(t, cdc.MustMarshalBinaryBare(pubKey), pubKey.Bytes())
}
