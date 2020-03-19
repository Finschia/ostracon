package vrf

import (
    "math/big"

    "github.com/tendermint/tendermint/crypto/ed25519"
)

var defaultVrf VrfEd25519

type Proof []byte

type Output []byte

func (op Output) ToInt() *big.Int {
    i := big.Int{}
    i.SetBytes(op)
    return &i
}

type VrfEd25519 interface {
    Prove(privateKey ed25519.PrivKey, message []byte) (Proof, error)
    Verify(publicKey ed25519.PubKey, proof Proof, message []byte) (bool, error)
    ProofToHash(proof Proof) (Output, error)
}

func Prove(privateKey ed25519.PrivKey, message []byte) (Proof, error) {
    return defaultVrf.Prove(privateKey, message)
}

func Verify(publicKey ed25519.PubKey, proof Proof, message []byte) (bool, error) {
    return defaultVrf.Verify(publicKey, proof, message)
}

func ProofToHash(proof Proof) (Output, error) {
    return defaultVrf.ProofToHash(proof)
}
