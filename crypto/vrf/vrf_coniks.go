package vrf

import (
	"bytes"
	"errors"

	coniksimpl "github.com/coniks-sys/coniks-go/crypto/vrf"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

type VrfEd25519Coniks struct {
	generatedHash []byte
	generatedProof []byte
}

func NewVrfEd25519Coniks() *VrfEd25519Coniks {
	return &VrfEd25519Coniks{nil, nil}
}

func NewVrfEd25519ConiksForVerifier(output Output, proof Proof) *VrfEd25519Coniks {
	return &VrfEd25519Coniks{output, proof}
}

func (base *VrfEd25519Coniks) Prove(privateKey ed25519.PrivKeyEd25519, message []byte) (Proof, error) {
	if len(privateKey) != coniksimpl.PrivateKeySize {
		return nil, errors.New("private key size is invalid")
	}
	coniksPrivKey := coniksimpl.PrivateKey(make([]byte, coniksimpl.PrivateKeySize))
	copy(coniksPrivKey, privateKey[:])
	hash, proof := coniksPrivKey.Prove(message)
	base.generatedHash = hash
	base.generatedProof = proof
	return proof, nil
}

func (base *VrfEd25519Coniks) Verify(publicKey ed25519.PubKeyEd25519, proof Proof, message []byte) (bool, error) {
	if base.generatedHash == nil {
		return false, errors.New("vrf hash was not given")
	}
	if bytes.Compare(base.generatedProof, proof) != 0 {
		return false, errors.New("proof is not same to the previously generated proof")
	}
	if len(publicKey) != coniksimpl.PublicKeySize {
		return false, errors.New("public key size is invalid")
	}
	coniksPubKey := coniksimpl.PublicKey(make([]byte, coniksimpl.PublicKeySize))
	copy(coniksPubKey, publicKey[:])
	return coniksPubKey.Verify(message, base.generatedHash, proof), nil
}

func (base *VrfEd25519Coniks) ProofToHash(proof Proof) (Output, error) {
	if base.generatedHash == nil {
		return nil, errors.New("vrf hash was not given")
	}
	if bytes.Compare(base.generatedProof, proof) != 0 {
		return nil, errors.New("proof is not same to the previously generated proof")
	}
	return base.generatedHash, nil
}
