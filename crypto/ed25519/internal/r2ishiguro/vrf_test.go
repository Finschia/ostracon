package r2ishiguro_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Finschia/ostracon/crypto/ed25519"
	"github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro"
	"github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro/testutil"
)

func TestVerify(t *testing.T) {
	privKey := ed25519.GenPrivKey()
	pubKey := privKey.PubKey().Bytes()
	message := []byte("hello, world")

	proof, err := testutil.Prove(privKey, message)
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
			valid, _ := r2ishiguro.Verify(pubKey, proof, tc.message)
			require.Equal(t, tc.valid, valid)
		})
	}
}

func TestProofToHash(t *testing.T) {
	privKey := ed25519.GenPrivKey()
	message := []byte("hello, world")

	t.Run("to hash r2ishiguro proof", func(t *testing.T) {
		proof, err := testutil.Prove(privKey, message)
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
	privKey := ed25519.GenPrivKey()
	pubKey := privKey.PubKey().Bytes()
	message := []byte("hello, world")

	var proof []byte
	var err error
	b.Run("VRF prove", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			proof, err = testutil.Prove(privKey, message)
		}
	})
	require.NoError(b, err)
	b.Run("VRF verify", func(b *testing.B) {
		b.ResetTimer()
		isValid, _ := r2ishiguro.Verify(pubKey, proof, message)
		if !isValid {
			err = fmt.Errorf("invalid")
		} else {
			err = nil
		}
	})
	require.NoError(b, err)
}
