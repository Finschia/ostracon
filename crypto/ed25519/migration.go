package ed25519

import (
	"fmt"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
	voivrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"

	r2vrf "github.com/Finschia/ostracon/crypto/ed25519/internal/r2ishiguro"
)

// vrf w/o prove
type vrfNoProve interface {
	Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte)
	ProofToHash(proof []byte) ([]byte, error)
}

var vrfs = map[int]vrfNoProve{
	voivrf.ProofSize: &voi{},
	r2vrf.ProofSize:  &r2{},
}

func getVrf(proof []byte) (vrfNoProve, error) {
	proofSize := len(proof)
	if vrf, exists := vrfs[proofSize]; exists {
		return vrf, nil
	}
	return nil, fmt.Errorf("invalid proof size: %d", proofSize)
}

func VRFVerify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	vrf, err := getVrf(proof)
	if err != nil {
		return false, nil
	}

	return vrf.Verify(pubKey, proof, message)
}

func ProofToHash(proof []byte) ([]byte, error) {
	vrf, err := getVrf(proof)
	if err != nil {
		return nil, err
	}

	return vrf.ProofToHash(proof)
}

// ValidateProof returns an error if the proof is not empty, but its
// size != vrf.ProofSize.
func ValidateProof(h []byte) error {
	if len(h) > 0 {
		if _, err := getVrf(h); err != nil {
			return fmt.Errorf("expected size to be %d bytes, got %d bytes",
				voivrf.ProofSize,
				len(h),
			)
		}
	}
	return nil
}

// github.com/oasisprotocol/curve25519-voi
var _ vrfNoProve = (*voi)(nil)

type voi struct{}

func (_ voi) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	return voivrf.Verify(pubKey, proof, message)
}

func (_ voi) ProofToHash(proof []byte) ([]byte, error) {
	return voivrf.ProofToHash(proof)
}

// github.com/r2ishiguro/vrf
var _ vrfNoProve = (*r2)(nil)

type r2 struct{}

func (_ r2) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	return r2vrf.Verify(pubKey, proof, message)
}

func (_ r2) ProofToHash(proof []byte) ([]byte, error) {
	return r2vrf.ProofToHash(proof)
}
