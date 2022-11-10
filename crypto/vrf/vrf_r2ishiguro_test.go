//go:build !libsodium && !coniks
// +build !libsodium,!coniks

package vrf

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVrfEd25519R2ishiguro_ProofToHash(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	message := []byte("hello, world")

	vrfr2ishiguro := newVrfEd25519r2ishiguro()

	t.Run("to hash r2ishiguro proof", func(t *testing.T) {
		proof, err := vrfr2ishiguro.Prove(privateKey, message)
		require.NoError(t, err)
		require.NotNil(t, proof)

		output, err := vrfr2ishiguro.ProofToHash(proof)
		require.NoError(t, err)
		require.NotNil(t, output)
	})

	t.Run("to hash other algo proof", func(t *testing.T) {
		proof := []byte("proof of test")
		output, err := vrfr2ishiguro.ProofToHash(proof)
		require.Error(t, err)
		require.Nil(t, output)
	})
}

func TestProveAndVerifyR2ishiguroByCryptoEd25519(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)

	verified, err := proveAndVerify(t, privateKey, publicKey)
	//
	// verified when using crypto ed25519
	//
	require.NoError(t, err)
	require.True(t, verified)
}
