package testutil

import (
	"github.com/r2ishiguro/vrf/go/vrf_ed25519"

	"github.com/Finschia/ostracon/crypto"
	"github.com/Finschia/ostracon/crypto/ed25519"
)

func Prove(privateKey []byte, message []byte) (crypto.Proof, error) {
	publicKey := ed25519.PrivKey(privateKey).PubKey().Bytes()
	return vrf_ed25519.ECVRF_prove(publicKey, privateKey, message)
}
