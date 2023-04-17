//go:build libsodium
// +build libsodium

package vrf

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"

	libsodium "github.com/Finschia/ostracon/crypto/vrf/internal/vrf"
)

func TestProveAndVerifyLibsodiumByCryptoEd25519(t *testing.T) {
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

func TestProveAndVerifyLibsodiumByLibsodiumEd25519(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	publicKey, privateKey := libsodium.KeyPairFromSeed(&secret)

	verified, err := proveAndVerify(t, privateKey[:], publicKey[:])
	//
	// verified when using libsodium ed25519
	//
	require.NoError(t, err)
	require.True(t, verified)
}
