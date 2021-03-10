package vrf

import (
	r2ishiguro "github.com/r2ishiguro/vrf/go/vrf_ed25519"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

type VrfEd25519r2ishiguro struct {

}

func NewVrfEd25519r2ishiguro() VrfEd25519r2ishiguro {
	return VrfEd25519r2ishiguro{}
}

func (base VrfEd25519r2ishiguro) Prove(privateKey ed25519.PrivKeyEd25519, message []byte) (Proof, error) {
	pubKey := privateKey.PubKey().(ed25519.PubKeyEd25519)
	return r2ishiguro.ECVRF_prove(pubKey[:], privateKey[:], message)
}

func (base VrfEd25519r2ishiguro) Verify(publicKey ed25519.PubKeyEd25519, proof Proof, message []byte) (bool, error) {
	return r2ishiguro.ECVRF_verify(publicKey[:], proof, message)
}

func (base VrfEd25519r2ishiguro) ProofToHash(proof Proof) (Output, error) {
	return r2ishiguro.ECVRF_proof2hash(proof), nil
}
