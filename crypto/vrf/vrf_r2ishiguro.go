// +build !libsodium

package vrf

import (
	r2ishiguro "github.com/r2ishiguro/vrf/go/vrf_ed25519"
)

type vrfEd25519r2ishiguro struct {
}

func init() {
	defaultVrf = newVrfEd25519r2ishiguro()
}

func newVrfEd25519r2ishiguro() vrfEd25519r2ishiguro {
	return vrfEd25519r2ishiguro{}
}

func (base vrfEd25519r2ishiguro) Prove(privateKey []byte, publicKey []byte, message []byte) (Proof, error) {
	return r2ishiguro.ECVRF_prove(publicKey, privateKey, message)
}

func (base vrfEd25519r2ishiguro) Verify(publicKey []byte, proof Proof, message []byte) (bool, error) {
	return r2ishiguro.ECVRF_verify(publicKey, proof, message)
}

func (base vrfEd25519r2ishiguro) ProofToHash(proof Proof) (Output, error) {
	return r2ishiguro.ECVRF_proof2hash(proof), nil
}
