//go:build libsodium
// +build libsodium

package vrf

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coniks "github.com/coniks-sys/coniks-go/crypto/vrf"
	ed25519consensus "github.com/hdevalence/ed25519consensus"
	libsodium "github.com/line/ostracon/crypto/vrf/internal/vrf"
	xed25519 "golang.org/x/crypto/ed25519"
)

var secret [SEEDBYTES]byte
var message []byte = []byte("hello, world")

func TestKeygen(t *testing.T) {
	for i := 0; i < SEEDBYTES; i++ {
		secret[i] = uint8(i)
	}
	privateKey, publicKey := keygenByEd25519(secret)

	t.Run("ed25519 and x/ed25519 have compatibility",
		func(t *testing.T) {
			testKeygenCompatibility(t, privateKey, publicKey,
				keygenByXed25519, require.Equal, require.Equal)
		})
	t.Run("ed25519 and coniks have NOT compatibility",
		func(t *testing.T) {
			testKeygenCompatibility(t, privateKey, publicKey,
				keygenByConiksEd25519, require.NotEqual, require.NotEqual)
		})
	t.Run("ed25519 and libsodium have compatibility",
		func(t *testing.T) {
			testKeygenCompatibility(t, privateKey, publicKey,
				keygenByLibsodiumEd25519, require.Equal, require.Equal)
		})
}

func testKeygenCompatibility(
	t *testing.T,
	sk1 ed25519.PrivateKey, pk1 ed25519.PublicKey,
	keyGen2 func(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey),
	skTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
	pkTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
) {
	sk2, pk2 := keyGen2(secret)
	skTest(t, sk1, sk2)
	pkTest(t, pk1, pk2)
}

func keygenByEd25519(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return privateKey, publicKey
}

func keygenByXed25519(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	publicKey, privateKey, _ := xed25519.GenerateKey(bytes.NewReader(secret[:]))
	return privateKey, publicKey
}

func keygenByConiksEd25519(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	privKey, _ := coniks.GenerateKey(bytes.NewReader(secret[:]))
	pubKey, _ := privKey.Public()
	privateKey := make([]byte, coniks.PrivateKeySize)
	copy(privateKey, privKey[:])
	publicKey := make([]byte, coniks.PublicKeySize)
	copy(publicKey, pubKey[:])
	return privateKey, publicKey
}

func keygenByLibsodiumEd25519(secret [SEEDBYTES]byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	var seed [libsodium.SEEDBYTES]byte
	copy(seed[:], secret[:])
	pubKey, privKey := libsodium.KeyPairFromSeed(&seed)
	privateKey := make([]byte, libsodium.SECRETKEYBYTES)
	copy(privateKey, privKey[:])
	publicKey := make([]byte, libsodium.PUBLICKEYBYTES)
	copy(publicKey, pubKey[:])
	return privateKey, publicKey
}

func TestKeypair(t *testing.T) {
	privateKey, publicKey := keygenByEd25519(secret)

	t.Run("ed25519 and x/ed25519 have compatibility",
		func(t *testing.T) {
			testKeypairCompatibilityWithXed25519(t, privateKey, publicKey,
				require.EqualValues, require.EqualValues)
		})
	t.Run("ed25519 and coniks have NOT compatibility",
		func(t *testing.T) {
			testKeypairCompatibilityWithConiksEd25519(t, privateKey, publicKey,
				require.EqualValues, require.NotEqualValues)
		})
	t.Run("ed25519 and libsodium have compatibility",
		func(t *testing.T) {
			testKeypairCompatibilityWithLibsodiumEd25519(t, privateKey, publicKey,
				require.EqualValues, require.EqualValues)
		})
}

func testKeypairCompatibilityWithXed25519(
	t *testing.T,
	sk ed25519.PrivateKey, pk ed25519.PublicKey,
	toPkTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
	fromSkTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
) {
	pk1, sk1, _ := xed25519.GenerateKey(bytes.NewReader(secret[:]))

	sk2 := ed25519.PrivateKey(make([]byte, ed25519.PrivateKeySize))
	copy(sk2[:], sk1[:])
	pk2 := sk2.Public().(ed25519.PublicKey)

	toPkTest(t, pk1[:], pk2[:])

	copy(sk1[:], sk[:])
	pk1 = sk1.Public().(xed25519.PublicKey)

	fromSkTest(t, pk1[:], pk2[:])
}

func testKeypairCompatibilityWithConiksEd25519(
	t *testing.T,
	sk ed25519.PrivateKey, pk ed25519.PublicKey,
	toPkTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
	fromSkTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
) {
	sk1, _ := coniks.GenerateKey(bytes.NewReader(secret[:]))
	pk1, _ := sk1.Public()

	sk2 := ed25519.PrivateKey(make([]byte, ed25519.PrivateKeySize))
	copy(sk2[:], sk1[:])
	pk2 := sk2.Public().(ed25519.PublicKey)

	toPkTest(t, pk1[:], pk2[:])

	copy(sk1[:], sk[:])
	pk1, _ = sk1.Public()

	fromSkTest(t, pk1[:], pk2[:])
}

func testKeypairCompatibilityWithLibsodiumEd25519(
	t *testing.T,
	sk ed25519.PrivateKey, pk ed25519.PublicKey,
	toPkTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
	fromSkTest func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}),
) {
	var seed [libsodium.SEEDBYTES]byte
	copy(seed[:], secret[:])
	pk1, sk1 := libsodium.KeyPairFromSeed(&seed)

	sk2 := ed25519.PrivateKey(make([]byte, ed25519.PrivateKeySize))
	copy(sk2[:], sk1[:])
	pk2 := sk2.Public().(ed25519.PublicKey)

	toPkTest(t, pk1[:], pk2[:])

	copy(sk1[:], sk[:])
	pk1 = libsodium.SkToPk(sk1)

	fromSkTest(t, pk1[:], pk2[:])
}

func TestSignVerify(t *testing.T) {
	sk, pk := keygenByEd25519(secret)
	t.Run("ed25519( and ed25519consensus.Verify) and xed25119 have compatibility", func(t *testing.T) {
		pk1, sk1, _ := xed25519.GenerateKey(bytes.NewReader(secret[:]))

		signature := ed25519.Sign(sk, message)
		valid := xed25519.Verify(pk1, message, signature)
		assert.True(t, valid)

		signature = xed25519.Sign(sk1, message)
		valid = ed25519.Verify(pk, message, signature)
		assert.True(t, valid)

		valid = ed25519consensus.Verify(pk, message, signature)
		assert.True(t, valid)
	})
}

func BenchmarkKeyGenByEd25519(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keygenByEd25519(secret)
	}
}

func BenchmarkKeyGenByConiksEd25519(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keygenByConiksEd25519(secret)
	}
}

func BenchmarkKeyGenByLibsodiumEd25519(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keygenByLibsodiumEd25519(secret)
	}
}
