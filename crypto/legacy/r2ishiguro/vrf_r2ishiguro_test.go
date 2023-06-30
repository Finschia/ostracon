package r2ishiguro_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/r2ishiguro/vrf/go/vrf_ed25519"

	"github.com/Finschia/ostracon/crypto"
	"github.com/Finschia/ostracon/crypto/legacy/r2ishiguro"
)

func prove(privateKey []byte, message []byte) (crypto.Proof, error) {
	publicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	return vrf_ed25519.ECVRF_prove(publicKey, privateKey, message)
}

func TestVrfEd25519R2ishiguro_ProofToHash(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	message := []byte("hello, world")

	t.Run("to hash r2ishiguro proof", func(t *testing.T) {
		proof, err := prove(privateKey, message)
		require.NoError(t, err)
		require.NotNil(t, proof)

		output, err := r2ishiguro.ProofToHash(proof)
		require.NoError(t, err)
		require.NotNil(t, output)
	})

	t.Run("to hash other algo proof", func(t *testing.T) {
		proof := []byte("proof of test")
		output, err := r2ishiguro.ProofToHash(proof)
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
