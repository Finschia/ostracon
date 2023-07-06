package ed25519

import (
	"fmt"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
	voivrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"

	r2vrf "github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro"
)

// vrf w/o prove
type VrfNoProve interface {
	Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte)
	ProofToHash(proof []byte) ([]byte, error)
	ValidateProof(proof []byte) error
}

var globalVrf = NewVersionedVrfNoProve()

func VRFVerify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	return globalVrf.Verify(pubKey, proof, message)
}

func ProofToHash(proof []byte) ([]byte, error) {
	return globalVrf.ProofToHash(proof)
}

// ValidateProof returns an error if the proof is not empty, but its
// size != vrf.ProofSize.
func ValidateProof(h []byte) error {
	if len(h) > 0 {
		if err := globalVrf.ValidateProof(h); err != nil {
			return fmt.Errorf("expected size to be %d bytes, got %d bytes",
				voivrf.ProofSize,
				len(h),
			)
		}
	}
	return nil
}

// versioned vrf
var _ VrfNoProve = (*versionedVrfNoProve)(nil)

type versionedVrfNoProve struct {
	version            int
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

func (v *versionedVrfNoProve) getVrf(proof []byte) (VrfNoProve, error) {
	proofSize := len(proof)
	if version, exists := v.proofSizeToVersion[proofSize]; exists && version >= v.version {
		v.version = version
		return v.vrfs[version], nil
	}
	return nil, fmt.Errorf("invalid proof size: %d", proofSize)
}

func (v versionedVrfNoProve) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	vrf, err := v.getVrf(proof)
	if err != nil {
		return false, nil
	}
	return vrf.Verify(pubKey, proof, message)
}

func (v versionedVrfNoProve) ProofToHash(proof []byte) ([]byte, error) {
	vrf, err := v.getVrf(proof)
	if err != nil {
		return nil, err
	}
	return vrf.ProofToHash(proof)
}

func (v versionedVrfNoProve) ValidateProof(proof []byte) error {
	vrf, err := v.getVrf(proof)
	if err != nil {
		return err
	}
	return vrf.ValidateProof(proof)
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

func (_ voiVrfNoProve) ValidateProof(proof []byte) error {
	proofSize := len(proof)
	if proofSize != voivrf.ProofSize {
		return fmt.Errorf("invalid proof size: %d", proofSize)
	}
	return nil
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

func (_ r2VrfNoProve) ValidateProof(proof []byte) error {
	proofSize := len(proof)
	if proofSize != r2vrf.ProofSize {
		return fmt.Errorf("invalid proof size: %d", proofSize)
	}
	return nil
}
