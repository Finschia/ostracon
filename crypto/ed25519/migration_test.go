package ed25519_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Finschia/ostracon/crypto/ed25519"
	voivrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"

	r2vrf "github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro"
	r2vrftestutil "github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro/testutil"
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

func TestVersionControl(t *testing.T) {
	vrf := ed25519.NewVersionedVrfNoProve()

	privKey := ed25519.GenPrivKey()
	message := []byte("hello, world")

	// generate proofs
	oldProof, err := r2vrftestutil.Prove(privKey, message)
	require.NoError(t, err)
	newProof, err := privKey.VRFProve(message)
	require.NoError(t, err)

	// old one is valid for now
	_, err = vrf.ProofToHash(oldProof)
	require.NoError(t, err)

	// new one is valid
	_, err = vrf.ProofToHash(newProof)
	require.NoError(t, err)

	// old one is not valid anymore
	_, err = vrf.ProofToHash(oldProof)
	require.Error(t, err)
}
