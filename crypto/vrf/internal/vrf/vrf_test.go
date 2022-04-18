//go:build libsodium
// +build libsodium

package vrf

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

var (
	Message = []string{
		"0B3BE52BF10F431AB07A44E9F89BBDD886B5B177A08FD54066694213930C9B2E",
		"EB0068756CA1BA8A497055958A50A71AA11E7F9A3CA967F8B3F7D6AF4F67911E",
		"BC77D2E540543BE2112972706EE88B006471E385A1A39E9D11B47F787E2A49AA",
		"F67D0305ABC12664F9F037C55C92CED3FFD6CB5875364E6C4A221534D77B7566",
		"AB609319AFD5EDCE91B3540EF77D83D96688C46CCC55175D8A4E3801F6F17239",
		"0E3921D46CFC6CEBAD33558F1BA38447FC9B3AF0BA034C1FD1DD5481E04C8D54",
		"7D59D1B9B556CC9434A1F0E5350103F3D41BF4C846A1B967B4E3443BF153DF58",
		"C1952358B51634232B39FB2BE2E42105319CE812DFEBD9117CCE9A78F2E6BC44",
		"999228C220CF8BA79B9815E6DB5D2F3C52A73E6CC314DB147A1E6FBD0BCDCC96",
		"B91F62DBCCA98A4453E5DF5AFE2EC521179D400F58B0174237D8D990CDBEFB8A",
	}
)

func enc(s []byte) string {
	return hex.EncodeToString(s)
}

func TestConstants(t *testing.T) {
	require.Equal(t, uint32(32), PUBLICKEYBYTES)
	require.Equal(t, uint32(64), SECRETKEYBYTES)
	require.Equal(t, uint32(32), SEEDBYTES)
	require.Equal(t, uint32(64), OUTPUTBYTES)
	require.Equal(t, "ietfdraft03", PRIMITIVE)
}

func TestKeyPair(t *testing.T) {
	var pk, sk = KeyPair()
	require.Equal(t, PUBLICKEYBYTES, uint32(len(pk)))
	require.Equal(t, SECRETKEYBYTES, uint32(len(sk)))
}

func TestKeyPairFromSeed(t *testing.T) {
	pkStr := "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29"
	skStr := "0000000000000000000000000000000000000000000000000000000000000000" + pkStr
	var seed [SEEDBYTES]byte
	var pk, sk = KeyPairFromSeed(&seed)
	require.Equal(t, PUBLICKEYBYTES, uint32(len(pk)))
	require.Equal(t, pkStr, enc(pk[:]))
	require.Equal(t, SECRETKEYBYTES, uint32(len(sk)))
	require.Equal(t, skStr, enc(sk[:]))

	var message [0]byte
	var proof, err1 = Prove(sk, message[:])
	require.NoError(t, err1)
	require.Equal(t, PROOFBYTES, uint32(len(proof)))

	var output, err2 = ProofToHash(proof)
	require.NoError(t, err2)
	require.Equal(t, OUTPUTBYTES, uint32(len(output)))
}

func TestIsValidKey(t *testing.T) {

	// generated from KeyPair()
	var pk1, _ = KeyPair()
	require.True(t, IsValidKey(pk1))

	// generated from KeyPairFromSeed()
	var seed [SEEDBYTES]byte
	var pk2, _ = KeyPairFromSeed(&seed)
	require.True(t, IsValidKey(pk2))

	// zero
	var zero [PUBLICKEYBYTES]byte
	require.False(t, IsValidKey(&zero))

	// random bytes
	var random [PUBLICKEYBYTES]byte
	var rng = rand.New(rand.NewSource(0))
	rng.Read(random[:])
	require.False(t, IsValidKey(&zero))
}

func TestInternalProveAndVerify(t *testing.T) {
	message := []byte("hello, world")

	var zero [SEEDBYTES]byte
	var pk, sk = KeyPairFromSeed(&zero)
	require.Equal(t, PUBLICKEYBYTES, uint32(len(pk)))
	require.Equal(t, SECRETKEYBYTES, uint32(len(sk)))

	var proof, err1 = Prove(sk, message)
	require.NoError(t, err1)
	require.Equal(t, PROOFBYTES, uint32(len(proof)))

	var output, err2 = ProofToHash(proof)
	require.NoError(t, err2)
	require.Equal(t, OUTPUTBYTES, uint32(len(output)))

	var verified, err3 = Verify(pk, proof, message)
	require.NoError(t, err3)
	require.Equal(t, output[:], verified[:])

	// essentially, the private key for ed25519 could be any value at a point on the finite field.
	var invalidPrivateKey [SECRETKEYBYTES]byte
	for i := range invalidPrivateKey {
		invalidPrivateKey[i] = 0xFF
	}
	var _, err4 = Prove(&invalidPrivateKey, message)
	require.Error(t, err4)

	// unexpected public key for Verify()
	var zero3 [PUBLICKEYBYTES]byte
	var _, err5 = Verify(&zero3, proof, message)
	require.Error(t, err5)

	// unexpected proof for Verify()
	var zero4 [PROOFBYTES]byte
	var _, err6 = Verify(pk, &zero4, message)
	require.Error(t, err6)

	// unexpected message for Verify()
	var message2 = []byte("good-by world")
	var _, err7 = Verify(pk, proof, message2)
	require.Error(t, err7)

	// essentially, the proof for ed25519 could be any value at a point on the finite field.
	var invalidProof [PROOFBYTES]byte
	for i := range invalidProof {
		invalidProof[i] = 0xFF
	}
	var _, err8 = ProofToHash(&invalidProof)
	require.Error(t, err8)
}

func TestSkToPk(t *testing.T) {
	var zero [SEEDBYTES]byte
	var expected, sk = KeyPairFromSeed(&zero)
	var actual = SkToPk(sk)
	require.Equal(t, expected[:], actual[:])
}

func TestSkToSeed(t *testing.T) {
	var zero [SEEDBYTES]byte
	var _, sk = KeyPairFromSeed(&zero)
	var actual = SkToSeed(sk)
	require.Equal(t, zero[:], actual[:])
}
