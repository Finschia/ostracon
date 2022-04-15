//go:build coniks
// +build coniks

package vrf

import (
	"bytes"
	"testing"

	"crypto/ed25519"

	coniks "github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/stretchr/testify/require"
)

func TestProveAndVerifyConiksByCryptoEd25519(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)

	verified, err := proveAndVerify(t, privateKey, publicKey)
	//
	// "un-verified" when using crypto ed25519
	// If you want to use coniks, you should use coniks ed25519
	//
	require.NoError(t, err)
	require.False(t, verified)
}

func TestProveAndVerifyConiksByConiksEd25519(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey, _ := coniks.GenerateKey(bytes.NewReader(secret[:]))
	publicKey, _ := privateKey.Public()

	verified, err := proveAndVerify(t, privateKey, publicKey)
	//
	// verified when using coniks ed25519
	//
	require.NoError(t, err)
	require.True(t, verified)
}
