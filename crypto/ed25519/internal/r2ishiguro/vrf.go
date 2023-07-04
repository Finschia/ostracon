package r2ishiguro

import (
	"github.com/r2ishiguro/vrf/go/vrf_ed25519"
)

const (
	ProofSize = 81
)

func Verify(publicKey []byte, proof []byte, message []byte) (bool, []byte) {
	isValid, err := vrf_ed25519.ECVRF_verify(publicKey, proof, message)
	if err != nil || !isValid {
		return false, nil
	}

	hash, err := ProofToHash(proof)
	if err != nil {
		return false, nil
	}

	return true, hash
}

func ProofToHash(proof []byte) ([]byte, error) {
	// validate proof with ECVRF_decode_proof
	_, _, _, err := vrf_ed25519.ECVRF_decode_proof(proof)
	if err != nil {
		return nil, err
	}
	return vrf_ed25519.ECVRF_proof2hash(proof), nil
}
