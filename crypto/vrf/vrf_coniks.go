package vrf

import (
	"bytes"
	"errors"

	coniksimpl "github.com/coniks-sys/coniks-go/crypto/vrf"
)

//nolint
type vrfEd25519Coniks struct {
	generatedHash  []byte
	generatedProof []byte
}

//nolint
func newVrfEd25519Coniks() *vrfEd25519Coniks {
	return &vrfEd25519Coniks{nil, nil}
}

//nolint
func newVrfEd25519ConiksForVerifier(output Output, proof Proof) *vrfEd25519Coniks {
	return &vrfEd25519Coniks{output, proof}
}

func (base *vrfEd25519Coniks) Prove(privateKey []byte, message []byte) (Proof, error) {
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

func (base *vrfEd25519Coniks) Verify(publicKey []byte, proof Proof, message []byte) (bool, error) {
	if base.generatedHash == nil {
		return false, errors.New("vrf hash was not given")
	}
	if !bytes.Equal(base.generatedProof, proof) {
		return false, errors.New("proof is not same to the previously generated proof")
	}
	if len(publicKey) != coniksimpl.PublicKeySize {
		return false, errors.New("public key size is invalid")
	}
	coniksPubKey := coniksimpl.PublicKey(make([]byte, coniksimpl.PublicKeySize))
	copy(coniksPubKey, publicKey[:])
	return coniksPubKey.Verify(message, base.generatedHash, proof), nil
}

func (base *vrfEd25519Coniks) ProofToHash(proof Proof) (Output, error) {
	if base.generatedHash == nil {
		return nil, errors.New("vrf hash was not given")
	}
	if !bytes.Equal(base.generatedProof, proof) {
		return nil, errors.New("proof is not same to the previously generated proof")
	}
	return base.generatedHash, nil
}
