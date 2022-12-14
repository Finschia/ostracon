//go:build coniks
// +build coniks

package vrf

import (
	"bytes"
	"errors"

	coniks "github.com/coniks-sys/coniks-go/crypto/vrf"
)

type vrfEd25519coniks struct {
	generatedHash  []byte
	generatedProof []byte
}

func init() {
	defaultVrf = newVrfEd25519coniks()
}

const (
	ProofSize  int = coniks.ProofSize
	OutputSize int = coniks.Size
)

func newVrfEd25519coniks() *vrfEd25519coniks {
	return &vrfEd25519coniks{nil, nil}
}

func newVrfEd25519coniksForVerifier(output Output, proof Proof) *vrfEd25519coniks {
	return &vrfEd25519coniks{output, proof}
}

func (base *vrfEd25519coniks) Prove(privateKey []byte, message []byte) (Proof, error) {
	if len(privateKey) != coniks.PrivateKeySize {
		return nil, errors.New("private key size is invalid")
	}
	coniksPrivKey := coniks.PrivateKey(make([]byte, coniks.PrivateKeySize))
	copy(coniksPrivKey, privateKey)
	hash, proof := coniksPrivKey.Prove(message)
	base.generatedHash = hash
	base.generatedProof = proof
	return proof, nil
}

func (base *vrfEd25519coniks) Verify(publicKey []byte, proof Proof, message []byte) (bool, error) {
	if base.generatedHash == nil {
		return false, errors.New("vrf hash was not given")
	}
	if !bytes.Equal(base.generatedProof, proof) {
		return false, errors.New("proof is not same to the previously generated proof")
	}
	if len(publicKey) != coniks.PublicKeySize {
		return false, errors.New("public key size is invalid")
	}
	coniksPubKey := coniks.PublicKey(make([]byte, coniks.PublicKeySize))
	copy(coniksPubKey, publicKey)
	return coniksPubKey.Verify(message, base.generatedHash, proof), nil
}

func (base *vrfEd25519coniks) ProofToHash(proof Proof) (Output, error) {
	if base.generatedHash == nil {
		return nil, errors.New("vrf hash was not given")
	}
	if !bytes.Equal(base.generatedProof, proof) {
		return nil, errors.New("proof is not same to the previously generated proof")
	}
	return base.generatedHash, nil
}
