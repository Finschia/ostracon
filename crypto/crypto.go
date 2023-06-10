package crypto

import (
	"github.com/Finschia/ostracon/crypto/tmhash"
	"github.com/Finschia/ostracon/libs/bytes"
)

const (
	// AddressSize is the size of a pubkey address.
	AddressSize = tmhash.TruncatedSize
)

// An address is a []byte, but hex-encoded even in JSON.
// []byte leaves us the option to change the address length.
// Use an alias so Unmarshal methods (with ptr receivers) are available too.
type Address = bytes.HexBytes

func AddressHash(bz []byte) Address {
	return Address(tmhash.SumTruncated(bz))
}

// Proof represents the VRF Proof.
// It should be defined separately from Ed25519 VRF Proof to avoid circular import.
type Proof []byte
type Output []byte

type PubKey interface {
	Address() Address
	Bytes() []byte
	VerifySignature(msg []byte, sig []byte) bool
	VRFVerify(proof []byte, seed []byte) (Output, error) // TODO 🏺 rename to VerifyVRFProof to match VerifySignature
	Equals(PubKey) bool
	Type() string
}

type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) ([]byte, error)
	VRFProve(seed []byte) (Proof, error)
	PubKey() PubKey
	Equals(PrivKey) bool
	Type() string
}

type Symmetric interface {
	Keygen() []byte
	Encrypt(plaintext []byte, secret []byte) (ciphertext []byte)
	Decrypt(ciphertext []byte, secret []byte) (plaintext []byte, err error)
}
