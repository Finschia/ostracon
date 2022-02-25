//go:build libsodium
// +build libsodium

package vrf

import (
	"bytes"
	"crypto/ed25519"
	"github.com/stretchr/testify/require"
	"testing"

	coniks "github.com/coniks-sys/coniks-go/crypto/vrf"
	libsodium "github.com/line/ostracon/crypto/vrf/internal/vrf"
)

var secret [SEEDBYTES]byte
var message []byte = []byte("hello, world")

func keyGen_ed25519(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return privateKey, publicKey
}

func keyGen_coniks_ed25519(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	privKey, _ := coniks.GenerateKey(bytes.NewReader(secret[:]))
	pubKey, _ := privKey.Public()
	privateKey := make([]byte, coniks.PrivateKeySize)
	copy(privateKey, privKey[:])
	publicKey := make([]byte, coniks.PublicKeySize)
	copy(publicKey, pubKey[:])
	return privateKey, publicKey
}

func keyGen_libsodium_ed25519(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	var seed [libsodium.SEEDBYTES]byte
	copy(seed[:], secret[:])
	pubKey, privKey := libsodium.KeyPairFromSeed(&seed)
	privateKey := make([]byte, libsodium.SECRETKEYBYTES)
	copy(privateKey, privKey[:])
	publicKey := make([]byte, libsodium.PUBLICKEYBYTES)
	copy(publicKey, pubKey[:])
	return privateKey, publicKey
}

func TestKeyGen(t *testing.T) {
	t.Logf("secret: [%s]", enc(secret[:]))

	privateKey, publicKey := keyGen_ed25519(secret)
	t.Logf("[ed25519  ]private key: [%s]", enc(privateKey[:]))
	t.Logf("[ed25519  ]public  key: [%s]", enc(publicKey[:]))

	coniksPrivateKey, coniksPublicKey := keyGen_coniks_ed25519(secret)
	t.Logf("[coniks   ]private key: [%s]", enc(coniksPrivateKey[:]))
	t.Logf("[coniks   ]public  key: [%s]", enc(coniksPublicKey[:]))

	libsodiumPrivateKey, libsodiumPublicKey := keyGen_libsodium_ed25519(secret)
	t.Logf("[libsodium]private key: [%s]", enc(libsodiumPrivateKey[:]))
	t.Logf("[libsodium]public  key: [%s]", enc(libsodiumPublicKey[:]))

	require.NotEqual(t, privateKey, coniksPrivateKey)
	require.NotEqual(t, publicKey, coniksPublicKey)
	require.Equal(t, privateKey, libsodiumPrivateKey)
	require.Equal(t, publicKey, libsodiumPublicKey)
}

func BenchmarkKeyGenED25519(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyGen_ed25519(secret)
	}
}

func BenchmarkKeyGenConiksED25519(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyGen_coniks_ed25519(secret)
	}
}

func BenchmarkKeyGenLibsodiumED25519(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyGen_libsodium_ed25519(secret)
	}
}
