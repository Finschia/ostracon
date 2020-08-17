package vrf

import (
	"math/big"
)

// defaultVrf is assigned to vrfEd25519r2ishiguro by init() of vrf_r2ishguro.go
// If you want to use libsodium for vrf implementation, then you should put build option like this
// `make build LIBSODIUM=1`
// Please refer https://github.com/line/tendermint/pull/41 for more detail
var defaultVrf vrfEd25519

type Proof []byte
type Output []byte

func (proof Proof) ToHash() ([]byte, error) {
	return ProofToHash(proof)
}

type vrfEd25519 interface {
	Prove(privateKey []byte, publicKey []byte, message []byte) (Proof, error)
	Verify(publicKey []byte, proof Proof, message []byte) (bool, error)
	ProofToHash(proof Proof) (Output, error)
}

func (op Output) ToInt() *big.Int {
	i := big.Int{}
	i.SetBytes(op)
	return &i
}

func Prove(privateKey []byte, publicKey []byte, message []byte) (Proof, error) {
	return defaultVrf.Prove(privateKey, publicKey, message)
}

func Verify(publicKey []byte, proof Proof, message []byte) (bool, error) {
	return defaultVrf.Verify(publicKey, proof, message)
}

func ProofToHash(proof Proof) (Output, error) {
	return defaultVrf.ProofToHash(proof)
}
