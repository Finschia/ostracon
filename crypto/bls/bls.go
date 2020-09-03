package bls

import (
	"bytes"
	"crypto/sha512"
	"crypto/subtle"
	"fmt"

	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/herumi/bls-eth-go-binary/bls"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

var _ crypto.PrivKey = PrivKeyBLS12{}

const (
	PrivKeyName      = "tendermint/PrivKeyBLS12"
	PubKeyName       = "tendermint/PubKeyBLS12"
	PrivKeyBLS12Size = 32
	PubKeyBLS12Size  = 48
	SignatureSize    = 96
	KeyType          = "bls12-381"
)

func init() {
	tmjson.RegisterType(PubKeyBLS12{}, PubKeyName)
	tmjson.RegisterType(PrivKeyBLS12{}, PrivKeyName)

	err := bls.Init(bls.BLS12_381)
	if err != nil {
		panic(fmt.Sprintf("ERROR: %s", err))
	}
	err = bls.SetETHmode(bls.EthModeLatest)
	if err != nil {
		panic(fmt.Sprintf("ERROR: %s", err))
	}
}

// PrivKeyBLS12 implements crypto.PrivKey.
type PrivKeyBLS12 [PrivKeyBLS12Size]byte

// AddSignature adds a BLS signature to the init. When the init is nil, then a new aggregate signature is built
// from specified signature.
func AddSignature(init []byte, signature []byte) (aggrSign []byte, err error) {
	if init == nil {
		blsSign := bls.Sign{}
		init = blsSign.Serialize()
	} else if len(init) != SignatureSize {
		err = fmt.Errorf("invalid BLS signature: aggregated signature size %d is not valid size %d",
			len(init), SignatureSize)
		return
	}
	if len(signature) != SignatureSize {
		err = fmt.Errorf("invalid BLS signature: signature size %d is not valid size %d",
			len(signature), SignatureSize)
		return
	}
	blsSign := bls.Sign{}
	err = blsSign.Deserialize(signature)
	if err != nil {
		return
	}
	aggrBLSSign := bls.Sign{}
	err = aggrBLSSign.Deserialize(init)
	if err != nil {
		return
	}
	aggrBLSSign.Add(&blsSign)
	aggrSign = aggrBLSSign.Serialize()
	return
}

func VerifyAggregatedSignature(aggregatedSignature []byte, pubKeys []PubKeyBLS12, msgs [][]byte) error {
	if len(pubKeys) != len(msgs) {
		return fmt.Errorf("the number of public keys %d doesn't match the one of messages %d",
			len(pubKeys), len(msgs))
	}
	if aggregatedSignature == nil {
		if len(pubKeys) == 0 {
			return nil
		}
		return fmt.Errorf(
			"the aggregate signature was omitted, even though %d public keys were specified", len(pubKeys))
	}
	aggrSign := bls.Sign{}
	err := aggrSign.Deserialize(aggregatedSignature)
	if err != nil {
		return err
	}
	blsPubKeys := make([]bls.PublicKey, len(pubKeys))
	hashes := make([][]byte, len(msgs))
	for i := 0; i < len(pubKeys); i++ {
		blsPubKeys[i] = bls.PublicKey{}
		err = blsPubKeys[i].Deserialize(pubKeys[i][:])
		if err != nil {
			return err
		}
		hash := sha512.Sum512_256(msgs[i])
		hashes[i] = hash[:]
	}
	if !aggrSign.VerifyAggregateHashes(blsPubKeys, hashes) {
		return fmt.Errorf("failed to verify the aggregated hashes by %d public keys", len(blsPubKeys))
	}
	return nil
}

// GenPrivKey generates a new BLS12-381 private key.
func GenPrivKey() PrivKeyBLS12 {
	sigKey := bls.SecretKey{}
	sigKey.SetByCSPRNG()
	sigKeyBinary := PrivKeyBLS12{}
	binary := sigKey.Serialize()
	if len(binary) != PrivKeyBLS12Size {
		panic(fmt.Sprintf("unexpected BLS private key size: %d != %d", len(binary), PrivKeyBLS12Size))
	}
	copy(sigKeyBinary[:], binary)
	return sigKeyBinary
}

// Bytes marshals the privkey using amino encoding.
func (privKey PrivKeyBLS12) Bytes() []byte {
	return privKey[:]
}

// Sign produces a signature on the provided message.
func (privKey PrivKeyBLS12) Sign(msg []byte) ([]byte, error) {
	if msg == nil {
		panic(fmt.Sprintf("Nil specified as the message"))
	}
	blsKey := bls.SecretKey{}
	err := blsKey.Deserialize(privKey[:])
	if err != nil {
		return nil, err
	}
	hash := sha512.Sum512_256(msg)
	sign := blsKey.SignHash(hash[:])
	return sign.Serialize(), nil
}

// VRFProve is not supported in BLS12.
func (privKey PrivKeyBLS12) VRFProve(seed []byte) (crypto.Proof, error) {
	return nil, fmt.Errorf("VRF prove is not supported by the BLS12")
}

// PubKey gets the corresponding public key from the private key.
func (privKey PrivKeyBLS12) PubKey() crypto.PubKey {
	blsKey := bls.SecretKey{}
	err := blsKey.Deserialize(privKey[:])
	if err != nil {
		panic(fmt.Sprintf("Not a BLS12-381 private key: %X", privKey[:]))
	}
	pubKey := blsKey.GetPublicKey()
	pubKeyBinary := PubKeyBLS12{}
	binary := pubKey.Serialize()
	if len(binary) != PubKeyBLS12Size {
		panic(fmt.Sprintf("unexpected BLS public key size: %d != %d", len(binary), PubKeyBLS12Size))
	}
	copy(pubKeyBinary[:], binary)
	return pubKeyBinary
}

// Equals - you probably don't need to use this.
// Runs in constant time based on length of the keys.
func (privKey PrivKeyBLS12) Equals(other crypto.PrivKey) bool {
	if otherEd, ok := other.(PrivKeyBLS12); ok {
		return subtle.ConstantTimeCompare(privKey[:], otherEd[:]) == 1
	}
	return false
}

// Type returns information to identify the type of this key.
func (privKey PrivKeyBLS12) Type() string {
	return KeyType
}

var _ crypto.PubKey = PubKeyBLS12{}

// PubKeyBLS12 implements crypto.PubKey for the BLS12-381 signature scheme.
type PubKeyBLS12 [PubKeyBLS12Size]byte

// Address is the SHA256-20 of the raw pubkey bytes.
func (pubKey PubKeyBLS12) Address() crypto.Address {
	return tmhash.SumTruncated(pubKey[:])
}

// Bytes marshals the PubKey using amino encoding.
func (pubKey PubKeyBLS12) Bytes() []byte {
	return pubKey[:]
}

func (pubKey PubKeyBLS12) VerifySignature(msg []byte, sig []byte) bool {
	// make sure we use the same algorithm to sign
	if len(sig) != SignatureSize {
		return false
	}
	blsPubKey := bls.PublicKey{}
	err := blsPubKey.Deserialize(pubKey[:])
	if err != nil {
		return false
	}
	blsSign := bls.Sign{}
	err = blsSign.Deserialize(sig)
	if err != nil {
		fmt.Printf("Signature Deserialize failed: %s", err)
		return false
	}
	hash := sha512.Sum512_256(msg)
	return blsSign.VerifyHash(&blsPubKey, hash[:])
}

// VRFVerify is not supported in BLS12.
func (pubKey PubKeyBLS12) VRFVerify(proof crypto.Proof, seed []byte) (crypto.Output, error) {
	return nil, fmt.Errorf("VRF verify is not supported by the BLS12")
}

func (pubKey PubKeyBLS12) String() string {
	return fmt.Sprintf("PubKeyBLS12{%X}", pubKey[:])
}

// nolint: golint
func (pubKey PubKeyBLS12) Equals(other crypto.PubKey) bool {
	if otherEd, ok := other.(PubKeyBLS12); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	}
	return false
}

// Type returns information to identify the type of this key.
func (pubKey PubKeyBLS12) Type() string {
	return KeyType
}
