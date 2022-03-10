//go:build !libsodium && !coniks
// +build !libsodium,!coniks

package vrf

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProveAndVerify_r2ishiguro_by_crypto_ed25519(t *testing.T) {
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
