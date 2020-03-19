// +build !libsodium

package vrf

import (
	r2ishiguro "github.com/r2ishiguro/vrf/go/vrf_ed25519"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

type VrfEd25519r2ishiguro struct {

}

func init() {
	defaultVrf = NewVrfEd25519r2ishiguro()
}

func NewVrfEd25519r2ishiguro() VrfEd25519r2ishiguro {
	return VrfEd25519r2ishiguro{}
}

func (base VrfEd25519r2ishiguro) Prove(privateKey ed25519.PrivKey, message []byte) (Proof, error) {
	pubKey := privateKey.PubKey().(ed25519.PubKey)
	return r2ishiguro.ECVRF_prove(pubKey[:], privateKey[:], message)
}

func (base VrfEd25519r2ishiguro) Verify(publicKey ed25519.PubKey, proof Proof, message []byte) (bool, error) {
	return r2ishiguro.ECVRF_verify(publicKey[:], proof, message)
}

func (base VrfEd25519r2ishiguro) ProofToHash(proof Proof) (Output, error) {
	return r2ishiguro.ECVRF_proof2hash(proof), nil
}
