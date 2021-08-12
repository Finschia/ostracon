// +build libsodium

// This libsodiumwrap package makes the VRF API in Algorand's libsodium C library available to golang.

package vrf

import (
	"bytes"
	libsodium "github.com/line/ostracon/crypto/vrf/internal/vrf"
)

type vrfEd25519libsodium struct {
}

func init() {
	defaultVrf = newVrfEd25519libsodium()
}

func newVrfEd25519libsodium() vrfEd25519 {
	return vrfEd25519libsodium{}
}

func (base vrfEd25519libsodium) Prove(privateKey []byte, message []byte) (Proof, error) {
	var privKey [libsodium.SECRETKEYBYTES]byte
	copy(privKey[:], privateKey)
	pf, err := libsodium.Prove(&privKey, message)
	if err != nil {
		return nil, err
	}
	return newProof(pf), nil
}

func (base vrfEd25519libsodium) Verify(publicKey []byte, proof Proof, message []byte) (bool, error) {
	var pubKey [libsodium.PUBLICKEYBYTES]byte
	copy(pubKey[:], publicKey)
	op, err := libsodium.Verify(&pubKey, toArray(proof), message)
	if err != nil {
		return false, err
	}
	hash, err := base.ProofToHash(proof)
	if err != nil {
		return false, err
	}
	return bytes.Compare(op[:], hash) == 0, nil
}

func (base vrfEd25519libsodium) ProofToHash(proof Proof) (Output, error) {
	op, err := libsodium.ProofToHash(toArray(proof))
	if err != nil {
		return nil, err
	}
	return newOutput(op), nil
}

func newProof(bytes *[libsodium.PROOFBYTES]byte) Proof {
	proof := make([]byte, libsodium.PROOFBYTES)
	copy(proof, bytes[:])
	return proof
}

func toArray(pf Proof) *[libsodium.PROOFBYTES]byte {
	var array [libsodium.PROOFBYTES]byte
	copy(array[:], pf)
	return &array
}

func newOutput(bytes *[libsodium.OUTPUTBYTES]byte) Output {
	output := make([]byte, libsodium.OUTPUTBYTES)
	copy(output[:], bytes[:])
	return output
}
