package encoding

import (
	"testing"

	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/composite"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestPubKeyFromToProto(t *testing.T) {
	vrf := ed25519.GenPrivKey()
	signer := bls.GenPrivKey()
	sk := composite.NewPrivKeyComposite(signer, vrf)
	pk := sk.PubKey()
	pbPubKey, err := PubKeyToProto(pk)
	if err != nil {
		t.Fatalf("The public key could not be converted to a ProtocolBuffers format: %s; %+v", err, pk)
	}
	pk2, err := PubKeyFromProto(&pbPubKey)
	if err != nil {
		t.Fatalf("The public key could not be retrieved from a ProtocolBuffers format: %s; %+v", err, pbPubKey)
	}
	cpk, ok := pk2.(composite.PubKeyComposite)
	if !ok {
		t.Fatalf("The retrieved public key was not composite key: %+v", pk2)
	}
	if !cpk.Equals(pk) {
		t.Fatalf("The retrieved composite public key was not match: %+v != %+v", cpk, pk)
	}
}
