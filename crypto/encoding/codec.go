package encoding

import (
	"fmt"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/composite"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/json"
	pc "github.com/tendermint/tendermint/proto/tendermint/crypto"
)

func init() {
	json.RegisterType((*pc.PublicKey)(nil), "tendermint.crypto.PublicKey")
	json.RegisterType((*pc.PublicKey_Ed25519)(nil), "tendermint.crypto.PublicKey_Ed25519")
	json.RegisterType((*pc.PublicKey_Secp256K1)(nil), "tendermint.crypto.PublicKey_Secp256K1")
}

// PubKeyToProto takes crypto.PubKey and transforms it to a protobuf Pubkey
func PubKeyToProto(k crypto.PubKey) (pc.PublicKey, error) {
	var kp pc.PublicKey
	switch k := k.(type) {
	case composite.PubKey:
		sign, err := PubKeyToProto(k.SignKey)
		if err != nil {
			return kp, err
		}
		vrf, err := PubKeyToProto(k.VrfKey)
		if err != nil {
			return kp, err
		}
		kp = pc.PublicKey{
			Sum: &pc.PublicKey_Composite{
				Composite: &pc.CompositePublicKey{
					SignKey: &sign,
					VrfKey:  &vrf,
				},
			},
		}
	case bls.PubKey:
		kp = pc.PublicKey{
			Sum: &pc.PublicKey_Bls12{
				Bls12: k[:],
			},
		}
	case ed25519.PubKey:
		kp = pc.PublicKey{
			Sum: &pc.PublicKey_Ed25519{
				Ed25519: k,
			},
		}
	case secp256k1.PubKey:
		kp = pc.PublicKey{
			Sum: &pc.PublicKey_Secp256K1{
				Secp256K1: k,
			},
		}
	default:
		return kp, fmt.Errorf("toproto: key type %v is not supported", k)
	}
	return kp, nil
}

// PubKeyFromProto takes a protobuf Pubkey and transforms it to a crypto.Pubkey
func PubKeyFromProto(k *pc.PublicKey) (crypto.PubKey, error) {
	switch k := k.Sum.(type) {
	case *pc.PublicKey_Composite:
		var pk composite.PubKey
		sign, err := PubKeyFromProto(k.Composite.SignKey)
		if err != nil {
			return pk, err
		}
		vrf, err := PubKeyFromProto(k.Composite.VrfKey)
		if err != nil {
			return pk, err
		}
		pk = composite.PubKey{
			SignKey: sign,
			VrfKey:  vrf,
		}
		return pk, nil
	case *pc.PublicKey_Ed25519:
		if len(k.Ed25519) != ed25519.PubKeySize {
			return nil, fmt.Errorf("invalid size for PubKeyEd25519. Got %d, expected %d",
				len(k.Ed25519), ed25519.PubKeySize)
		}
		pk := make(ed25519.PubKey, ed25519.PubKeySize)
		copy(pk, k.Ed25519)
		return pk, nil
	case *pc.PublicKey_Bls12:
		if len(k.Bls12) != bls.PubKeySize {
			return nil, fmt.Errorf("invalid size for PubKeyBls12. Got %d, expected %d",
				len(k.Bls12), ed25519.PubKeySize)
		}
		pk := bls.PubKey{}
		copy(pk[:], k.Bls12)
		return pk, nil
	case *pc.PublicKey_Secp256K1:
		if len(k.Secp256K1) != secp256k1.PubKeySize {
			return nil, fmt.Errorf("invalid size for PubKeySecp256k1. Got %d, expected %d",
				len(k.Secp256K1), secp256k1.PubKeySize)
		}
		pk := make(secp256k1.PubKey, secp256k1.PubKeySize)
		copy(pk, k.Secp256K1)
		return pk, nil
	default:
		return nil, fmt.Errorf("fromproto: key type %v is not supported", k)
	}
}
