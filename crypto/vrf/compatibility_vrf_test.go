//go:build libsodium
// +build libsodium

package vrf

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"

	r2ishiguro "github.com/r2ishiguro/vrf/go/vrf_ed25519"
)

func TestProveAndVerifyCompatibility(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)

	sk := make([]byte, ed25519.PrivateKeySize)
	copy(sk, privateKey[:])
	pk := make([]byte, ed25519.PublicKeySize)
	copy(pk, publicKey[:])

	libsodiumImpl := newVrfEd25519libsodium()

	t.Run("libsodium.Prove and r2ishiguro.Verify have NOT compatibility", func(t *testing.T) {
		proof, err := libsodiumImpl.Prove(sk, message)
		require.NoError(t, err)
		require.NotNil(t, proof)

		valid, err := r2ishiguro.ECVRF_verify(pk, proof, message)
		require.Error(t, err)
		require.False(t, valid)
	})
	t.Run("r2ishiguro.Prove and libsodium.Verify have NOT compatibility", func(t *testing.T) {
		proof, err := r2ishiguro.ECVRF_prove(pk, sk, message)
		require.NoError(t, err)
		require.NotNil(t, proof)

		valid, err := libsodiumImpl.Verify(pk, proof, message)
		require.Error(t, err)
		require.False(t, valid)
	})
}
