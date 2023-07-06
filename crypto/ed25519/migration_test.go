package ed25519_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Finschia/ostracon/crypto/ed25519"
	voivrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"

	r2vrf "github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro"
)

func TestVerify(t *testing.T) {
	pubkey, message := []byte("pubkey"), []byte("message")
	valid, _ := ed25519.VRFVerify(pubkey, make([]byte, 1), message)
	require.False(t, valid)

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
			valid, _ := ed25519.NewVersionedVrfNoProve().Verify(pubkey, tc.proof, message)
			require.Equal(t, tc.valid, valid)
		})
	}
}

func TestProofToHash(t *testing.T) {
	_, err := ed25519.ProofToHash(make([]byte, 1))
	require.Error(t, err)

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
			_, err := ed25519.NewVersionedVrfNoProve().ProofToHash(tc.proof)
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateProof(t *testing.T) {
	err := ed25519.ValidateProof(make([]byte, 1))
	require.Error(t, err)
	err = ed25519.ValidateProof(make([]byte, 0))
	require.NoError(t, err)

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
			valid: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := ed25519.NewVersionedVrfNoProve().ValidateProof(tc.proof)
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestVersionControl(t *testing.T) {
	vrf := ed25519.NewVersionedVrfNoProve()

	// old one is valid for now
	oldProof := make([]byte, r2vrf.ProofSize)
	require.NoError(t, vrf.ValidateProof(oldProof))

	// new one is valid
	newProof := make([]byte, voivrf.ProofSize)
	require.NoError(t, vrf.ValidateProof(newProof))

	// old one is not valid anymore
	require.Error(t, vrf.ValidateProof(oldProof))
}
