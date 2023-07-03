package ed25519

import (
	"fmt"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
	voivrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"

	r2vrf "github.com/Finschia/ostracon/crypto/legacy/r2ishiguro"
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

// voi
var _ vrfNoProve = (*voi)(nil)

type voi struct{}

func (_ voi) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	return voivrf.Verify(pubKey, proof, message)
}

func (_ voi) ProofToHash(proof []byte) ([]byte, error) {
	return voivrf.ProofToHash(proof)
}

// r2ishiguro
var _ vrfNoProve = (*r2)(nil)

type r2 struct{}

func (_ r2) Verify(pubKey ed25519.PublicKey, proof []byte, message []byte) (bool, []byte) {
	return r2vrf.Verify(pubKey, proof, message)
}

func (_ r2) ProofToHash(proof []byte) ([]byte, error) {
	return r2vrf.ProofToHash(proof)
}
