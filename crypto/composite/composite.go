package composite

import (
	"bytes"
	"fmt"

	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/math"

	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/tmhash"

	"github.com/tendermint/tendermint/crypto"
)

// PubKeyComposite and PrivKeyComposite are intended to allow public key algorithms to be selected for each function.

const (
	PubKeyName  = "tendermint/PubKeyComposite"
	PrivKeyName = "tendermint/PrivKeyComposite"

	KeyType               = "composite"
	KeyTypeBlsWithEd25519 = KeyType + "(" + bls.KeyType + "," + ed25519.KeyType + ")"
)

var MaxSignatureSize = math.MaxInt(ed25519.SignatureSize, bls.SignatureSize)

func init() {
	tmjson.RegisterType(PubKeyComposite{}, PubKeyName)
	tmjson.RegisterType(PrivKeyComposite{}, PrivKeyName)
}

type PubKeyComposite struct {
	SignKey crypto.PubKey `json:"sign"`
	VrfKey  crypto.PubKey `json:"vrf"`
}

func (pk *PubKeyComposite) Identity() crypto.PubKey {
	return pk.VrfKey
}

func (pk PubKeyComposite) Address() crypto.Address {
	return crypto.Address(tmhash.SumTruncated(pk.Bytes()))
}

func (pk PubKeyComposite) Bytes() []byte {
	msg := bytes.NewBuffer(pk.SignKey.Bytes())
	msg.Write(pk.VrfKey.Bytes())
	return msg.Bytes()
}

func (pk PubKeyComposite) VerifySignature(msg []byte, sig []byte) bool {
	return pk.SignKey.VerifySignature(msg, sig)
}

// VRFVerify verifies that the given VRF Proof was generated from the seed by the owner of this public key.
func (pk PubKeyComposite) VRFVerify(proof crypto.Proof, seed []byte) (crypto.Output, error) {
	return pk.VrfKey.VRFVerify(proof, seed)
}

func (pk PubKeyComposite) Equals(key crypto.PubKey) bool {
	other, ok := key.(PubKeyComposite)
	return ok && pk.SignKey.Equals(other.SignKey) && pk.VrfKey.Equals(other.VrfKey)
}

func (pk PubKeyComposite) Type() string {
	return fmt.Sprintf("%s(%s,%s)", KeyType, pk.SignKey.Type(), pk.VrfKey.Type())
}

type PrivKeyComposite struct {
	SignKey crypto.PrivKey `json:"sign"`
	VrfKey  crypto.PrivKey `json:"vrf"`
}

func GenPrivKey() *PrivKeyComposite {
	return NewPrivKeyComposite(bls.GenPrivKey(), ed25519.GenPrivKey())
}

func NewPrivKeyComposite(sign crypto.PrivKey, vrf crypto.PrivKey) *PrivKeyComposite {
	return &PrivKeyComposite{SignKey: sign, VrfKey: vrf}
}

func (sk PrivKeyComposite) Identity() crypto.PrivKey {
	return sk.VrfKey
}

func (sk PrivKeyComposite) Bytes() []byte {
	return sk.Identity().Bytes()
}

func (sk PrivKeyComposite) Sign(msg []byte) ([]byte, error) {
	return sk.SignKey.Sign(msg)
}

// VRFProve generates a VRF Proof for given seed to generate a verifiable random.
func (sk PrivKeyComposite) VRFProve(seed []byte) (crypto.Proof, error) {
	return sk.VrfKey.VRFProve(seed)
}

func (sk PrivKeyComposite) PubKey() crypto.PubKey {
	return PubKeyComposite{sk.SignKey.PubKey(), sk.VrfKey.PubKey()}
}

func (sk PrivKeyComposite) Equals(key crypto.PrivKey) bool {
	switch other := key.(type) {
	case *PrivKeyComposite:
		return sk.SignKey.Equals(other.SignKey) && sk.VrfKey.Equals(other.VrfKey)
	default:
		return false
	}
}

func (sk PrivKeyComposite) Type() string {
	return fmt.Sprintf("%s(%s,%s)", KeyType, sk.SignKey.Type(), sk.VrfKey.Type())
}
