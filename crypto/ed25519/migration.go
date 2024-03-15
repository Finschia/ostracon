package ed25519

import (
	"fmt"
	"sync"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
	voivrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"

	r2vrf "github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro"
)

// vrf w/o prove
// vrf Prove() MUST use its latest implementation, while this allows
// to verify the old blocks.
type VrfNoProve interface {
	Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte)
	ProofToHash(proof []byte) ([]byte, error)
}

// following logics MUST use this instance:
// - VRFVerify()
// - ProofToHash()
var (
	globalVrf   = NewVersionedVrfNoProve()
	globalVrfMu = sync.Mutex{}
)

func VRFVerify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	globalVrfMu.Lock()
	defer globalVrfMu.Unlock()
	return globalVrf.Verify(pubKey, proof, message)
}

func ProofToHash(proof []byte) ([]byte, error) {
	globalVrfMu.Lock()
	defer globalVrfMu.Unlock()
	return globalVrf.ProofToHash(proof)
}

// ValidateProof returns an error if the proof is not empty, but its
// size != vrf.ProofSize.
func ValidateProof(h []byte) error {
	proofSize := len(h)
	if proofSize != voivrf.ProofSize && proofSize != r2vrf.ProofSize {
		return fmt.Errorf("expected size to be %d bytes, got %d bytes",
			voivrf.ProofSize,
			proofSize,
		)
	}
	return nil
}

// versioned vrf have all the implementations inside.
// it updates its version whenever it encounters the new proof format.
// it CANNOT downgrade its version.
var _ VrfNoProve = (*versionedVrfNoProve)(nil)

type versionedVrfNoProve struct {
	mu      sync.Mutex
	version int

	proofSizeToVersion map[int]int
	vrfs               map[int]VrfNoProve
}

func NewVersionedVrfNoProve() VrfNoProve {
	return &versionedVrfNoProve{
		version: 0,
		proofSizeToVersion: map[int]int{
			r2vrf.ProofSize:  0,
			voivrf.ProofSize: 1,
		},
		vrfs: map[int]VrfNoProve{
			0: &r2VrfNoProve{},
			1: &voiVrfNoProve{},
		},
	}
}

// getVersion emits error if the proof is old one.
func (v *versionedVrfNoProve) getVrf(proof []byte) (VrfNoProve, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	proofSize := len(proof)
	if version, exists := v.proofSizeToVersion[proofSize]; exists && version >= v.version {
		v.version = version
		return v.vrfs[version], nil
	}
	return nil, fmt.Errorf("invalid proof size: %d", proofSize)
}

func (v *versionedVrfNoProve) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	vrf, err := v.getVrf(proof)
	if err != nil {
		return false, nil
	}

	return vrf.Verify(pubKey, proof, message)
}

func (v *versionedVrfNoProve) ProofToHash(proof []byte) ([]byte, error) {
	vrf, err := v.getVrf(proof)
	if err != nil {
		return nil, err
	}
	return vrf.ProofToHash(proof)
}

// github.com/oasisprotocol/curve25519-voi
var _ VrfNoProve = (*voiVrfNoProve)(nil)

type voiVrfNoProve struct{}

func (_ voiVrfNoProve) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	return voivrf.Verify(pubKey, proof, message)
}

func (_ voiVrfNoProve) ProofToHash(proof []byte) ([]byte, error) {
	return voivrf.ProofToHash(proof)
}

// github.com/r2ishiguro/vrf
var _ VrfNoProve = (*r2VrfNoProve)(nil)

type r2VrfNoProve struct{}

func (_ r2VrfNoProve) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	return r2vrf.Verify(pubKey, proof, message)
}

func (_ r2VrfNoProve) ProofToHash(proof []byte) ([]byte, error) {
	return r2vrf.ProofToHash(proof)
}
