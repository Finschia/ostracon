package encoding

import (
	"reflect"
	"testing"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/composite"
)

func testPubKeyFromToProto(t *testing.T, sk crypto.PrivKey) {
	pk := sk.PubKey()
	pbPubKey, err := PubKeyToProto(pk)
	if err != nil {
		t.Fatalf("The public key could not be converted to a ProtocolBuffers format: %s; %+v", err, pk)
	}
	pk2, err := PubKeyFromProto(&pbPubKey)
	if err != nil {
		t.Fatalf("The public key could not be retrieved from a ProtocolBuffers format: %s; %+v", err, pbPubKey)
	}
	if reflect.TypeOf(pk2) != reflect.TypeOf(pk) {
		t.Fatalf("The retrieved public key was not %s key: %+v", reflect.TypeOf(pk), pk2)
	}
	if !pk2.Equals(pk) {
		t.Fatalf("The retrieved public key was not match: %+v != %+v", pk2, pk)
	}
}

func TestPubKeyFromToProto(t *testing.T) {
	testPubKeyFromToProto(t, ed25519.GenPrivKey())
	testPubKeyFromToProto(t, bls.GenPrivKey())
	testPubKeyFromToProto(t, composite.NewPrivKeyComposite(bls.GenPrivKey(), ed25519.GenPrivKey()))
}
