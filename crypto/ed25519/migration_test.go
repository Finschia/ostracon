package ed25519_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Finschia/ostracon/crypto/ed25519"
	voivrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"

	r2vrf "github.com/Finschia/ostracon/crypto/legacy/r2ishiguro"
)

func TestVRFVerify(t *testing.T) {
	cases := map[string]struct {
		proof []byte
		valid bool
	}{
		"invalid format": {
			proof: make([]byte, 1),
		},
		"voi invalid proof": {
			proof: make([]byte, voivrf.ProofSize),
		},
		"r2ishiguro invalid proof": {
			proof: make([]byte, r2vrf.ProofSize),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			pubkey, message := []byte("pubkey"), []byte("message")
			valid, _ := ed25519.VRFVerify(pubkey, tc.proof, message)
			require.Equal(t, tc.valid, valid)
		})
	}
}

func TestProofToHash(t *testing.T) {
	cases := map[string]struct {
		proof []byte
		valid bool
	}{
		"invalid format": {
			proof: make([]byte, 1),
		},
		"voi invalid proof": {
			proof: make([]byte, voivrf.ProofSize),
			valid: true,
		},
		"r2ishiguro proof": {
			proof: make([]byte, r2vrf.ProofSize),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ed25519.ProofToHash(tc.proof)
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
