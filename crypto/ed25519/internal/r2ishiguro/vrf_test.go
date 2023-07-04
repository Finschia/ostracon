package r2ishiguro_test

import (
	"crypto/ed25519"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/r2ishiguro/vrf/go/vrf_ed25519"

	"github.com/Finschia/ostracon/crypto"
	"github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro"
)

const (
	SEEDBYTES = ed25519.SeedSize
)

func prove(privateKey []byte, message []byte) (crypto.Proof, error) {
	publicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	return vrf_ed25519.ECVRF_prove(publicKey, privateKey, message)
}

func TestVerify(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)

	message := []byte("hello, world")
	proof, err := prove(privateKey, message)
	assert.NoError(t, err)
	assert.NotNil(t, proof)

	cases := map[string]struct {
		message []byte
		valid   bool
	}{
		"valid": {
			message: message,
			valid:   true,
		},
		"invalid": {
			message: []byte("deadbeef"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			valid, _ := r2ishiguro.Verify(publicKey, proof, tc.message)
			require.Equal(t, tc.valid, valid)
		})
	}
}

func TestProofToHash(t *testing.T) {
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

func BenchmarkProveAndVerify(b *testing.B) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)
	message := []byte("hello, world")

	var proof []byte
	var err error
	b.Run("VRF prove", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			proof, err = prove(privateKey, message)
		}
	})
	require.NoError(b, err)
	b.Run("VRF verify", func(b *testing.B) {
		b.ResetTimer()
		isValid, _ := r2ishiguro.Verify(publicKey, proof, message)
		if !isValid {
			err = fmt.Errorf("invalid")
		} else {
			err = nil
		}
	})
	require.NoError(b, err)
}
