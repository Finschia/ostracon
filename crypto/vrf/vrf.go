// This vrf package makes the VRF API in Algorand's libsodium C library available to golang.
package vrf

import (
    "github.com/tendermint/tendermint/crypto/ed25519"
    vrfimpl "github.com/tendermint/tendermint/crypto/vrf/internal/vrf"
    "math/big"
    "unsafe"
)

const (
    PROOFBYTES = vrfimpl.PROOFBYTES
    OUTPUTBYTES = vrfimpl.OUTPUTBYTES
)

type Proof [PROOFBYTES]byte

type Output [OUTPUTBYTES]byte

func newProof(bytes *[PROOFBYTES]byte) *Proof {
    proof := Proof{}
    copy(proof[:], bytes[:])
    return &proof
}

func (pf *Proof) toBytes() *[PROOFBYTES]byte {
    return (*[PROOFBYTES]byte)(unsafe.Pointer(pf))
}

func (pf *Proof) ToHash() (*Output, error) {
    op, err := vrfimpl.ProofToHash(pf.toBytes())
    if err != nil {
        return nil, err
    }
    return newOutput(op), nil
}

func newOutput(bytes *[OUTPUTBYTES]byte) *Output {
    output := Output{}
    copy(output[:], bytes[:])
    return &output
}

func (op *Output) ToInt() *big.Int {
    i := big.Int{}
    i.SetBytes(op[:])
    return &i
}

func Prove(privateKey *ed25519.PrivKey, message []byte) (*Proof, error) {
    privKey := (*[vrfimpl.SECRETKEYBYTES]byte)(unsafe.Pointer(privateKey))
    pf, err := vrfimpl.Prove(privKey, message)
    if err != nil {
        return nil, err
    }
    return newProof(pf), nil
}

func Verify(publicKey *ed25519.PubKey, proof *Proof, message []byte) (*Output, error) {
    pubKey := (*[vrfimpl.PUBLICKEYBYTES]byte)(unsafe.Pointer(publicKey))
    op, err := vrfimpl.Verify(pubKey, proof.toBytes(), message)
    if err != nil {
        return nil, err
    }
    return newOutput(op), nil
}
