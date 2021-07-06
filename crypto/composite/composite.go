package composite

import (
	"bytes"
	"fmt"

	tmjson "github.com/line/ostracon/libs/json"
	"github.com/line/ostracon/libs/math"

	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/bls"
	"github.com/line/ostracon/crypto/ed25519"
	"github.com/line/ostracon/crypto/tmhash"
)

// composite.PubKey and composite.PrivKey are intended to allow public key algorithms to be selected for each function.

const (
	PubKeyName  = "tendermint/PubKeyComposite"
	PrivKeyName = "tendermint/PrivKeyComposite"

	KeyType               = "composite"
	KeyTypeBlsWithEd25519 = KeyType + "(" + bls.KeyType + "," + ed25519.KeyType + ")"
)

var MaxSignatureSize = math.MaxInt(ed25519.SignatureSize, bls.SignatureSize)

func init() {
	tmjson.RegisterType(PubKey{}, PubKeyName)
	tmjson.RegisterType(PrivKey{}, PrivKeyName)
}

type PubKey struct {
	SignKey crypto.PubKey `json:"sign"`
	VrfKey  crypto.PubKey `json:"vrf"`
}

func (pk *PubKey) Identity() crypto.PubKey {
	return pk.VrfKey
}

func (pk PubKey) Address() crypto.Address {
	return crypto.Address(tmhash.SumTruncated(pk.Bytes()))
}

func (pk PubKey) Bytes() []byte {
	msg := bytes.NewBuffer(pk.SignKey.Bytes())
	msg.Write(pk.VrfKey.Bytes())
	return msg.Bytes()
}

func (pk PubKey) VerifySignature(msg []byte, sig []byte) bool {
	return pk.SignKey.VerifySignature(msg, sig)
}

// VRFVerify verifies that the given VRF Proof was generated from the seed by the owner of this public key.
func (pk PubKey) VRFVerify(proof crypto.Proof, seed []byte) (crypto.Output, error) {
	return pk.VrfKey.VRFVerify(proof, seed)
}

func (pk PubKey) Equals(key crypto.PubKey) bool {
	other, ok := key.(PubKey)
	return ok && pk.SignKey.Equals(other.SignKey) && pk.VrfKey.Equals(other.VrfKey)
}

func (pk PubKey) Type() string {
	return fmt.Sprintf("%s(%s,%s)", KeyType, pk.SignKey.Type(), pk.VrfKey.Type())
}

type PrivKey struct {
	SignKey crypto.PrivKey `json:"sign"`
	VrfKey  crypto.PrivKey `json:"vrf"`
}

func GenPrivKey() *PrivKey {
	return NewPrivKeyComposite(bls.GenPrivKey(), ed25519.GenPrivKey())
}

func NewPrivKeyComposite(sign crypto.PrivKey, vrf crypto.PrivKey) *PrivKey {
	return &PrivKey{SignKey: sign, VrfKey: vrf}
}

func (sk PrivKey) Identity() crypto.PrivKey {
	return sk.VrfKey
}

func (sk PrivKey) Bytes() []byte {
	return sk.Identity().Bytes()
}

func (sk PrivKey) Sign(msg []byte) ([]byte, error) {
	return sk.SignKey.Sign(msg)
}

// VRFProve generates a VRF Proof for given seed to generate a verifiable random.
func (sk PrivKey) VRFProve(seed []byte) (crypto.Proof, error) {
	return sk.VrfKey.VRFProve(seed)
}

func (sk PrivKey) PubKey() crypto.PubKey {
	return PubKey{sk.SignKey.PubKey(), sk.VrfKey.PubKey()}
}

func (sk PrivKey) Equals(key crypto.PrivKey) bool {
	switch other := key.(type) {
	case *PrivKey:
		return sk.SignKey.Equals(other.SignKey) && sk.VrfKey.Equals(other.VrfKey)
	default:
		return false
	}
}

func (sk PrivKey) Type() string {
	return fmt.Sprintf("%s(%s,%s)", KeyType, sk.SignKey.Type(), sk.VrfKey.Type())
}
