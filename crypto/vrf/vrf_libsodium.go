// This libsodiumwrap package makes the VRF API in Algorand's libsodium C library available to golang.

package vrf

import (
	"bytes"
	"unsafe"

	"github.com/tendermint/tendermint/crypto/ed25519"
	libsodium "github.com/tendermint/tendermint/crypto/vrf/internal/vrf"
)

type VrfImplLibsodium struct {
}

func NewVrfEd25519ImplLibsodium() VrfEd25519 {
	return VrfImplLibsodium{}
}

func (base VrfImplLibsodium) Prove(privateKey ed25519.PrivKeyEd25519, message []byte) (Proof, error) {
	privKey := (*[libsodium.SECRETKEYBYTES]byte)(unsafe.Pointer(&privateKey))
	pf, err := libsodium.Prove(privKey, message)
	if err != nil {
		return nil, err
	}
	return newProof(pf), nil
}

func (base VrfImplLibsodium) Verify(publicKey ed25519.PubKeyEd25519, proof Proof, message []byte) (bool, error) {
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

func (base VrfImplLibsodium) ProofToHash(proof Proof) (Output, error) {
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
