// +build libsodium

// This libsodiumwrap package makes the VRF API in Algorand's libsodium C library available to golang.

package vrf

import (
	"bytes"
	"unsafe"

	libsodium "github.com/line/ostracon/crypto/vrf/internal/vrf"
)

type vrfImplLibsodium struct {
}

func newVrfEd25519ImplLibsodium() vrfEd25519 {
	return vrfImplLibsodium{}
}

func init() {
	defaultVrf = newVrfEd25519ImplLibsodium()
}

func (base vrfImplLibsodium) Prove(privateKey []byte, message []byte) (Proof, error) {
	privKey := (*[libsodium.SECRETKEYBYTES]byte)(unsafe.Pointer(&(*privateKey)))
	pf, err := libsodium.Prove(privKey, message)
	if err != nil {
		return nil, err
	}
	return newProof(pf), nil
}

func (base vrfImplLibsodium) Verify(publicKey []byte, proof Proof, message []byte) (bool, error) {
	pubKey := (*[libsodium.PUBLICKEYBYTES]byte)(unsafe.Pointer(&publicKey))
	op, err := libsodium.Verify(pubKey, toArray(proof), message)
	if err != nil {
		return false, err
	}
	hash, err := base.ProofToHash(proof)
	if err != nil {
		return false, err
	}
	return bytes.Compare(op[:], hash) == 0, nil
}

func (base vrfImplLibsodium) ProofToHash(proof Proof) (Output, error) {
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
