package vrf

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	SEEDBYTES = ed25519.SeedSize
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

func TestProofToHash(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	message := []byte("hello, world")

	proof, err1 := Prove(privateKey, message)
	if err1 != nil {
		t.Fatalf("failed to prove: %s", err1)
	}

	_, err2 := ProofToHash(proof)
	if err2 != nil {
		t.Errorf("failed to convert to hash: %s", enc(proof[:]))
	}

	t.Skip("Invalid proof checking is available only for libsodium")
	// check to fail for invalid proof bytes
	for i := range proof {
		proof[i] = 0xFF
	}
	op3, err3 := ProofToHash(proof)
	if err3 == nil {
		t.Errorf("unexpected hash for invalid proof: %s", enc(op3[:]))
	}
}

func TestProve(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)

	t.Logf("private key: [%s]", enc(privateKey[:]))
	t.Logf("public  key: [%s]", enc(publicKey[:]))

	message := []byte("hello, world")
	proof, err1 := Prove(privateKey, message)
	if err1 != nil {
		t.Fatalf("failed to prove: %s", err1)
	}
	t.Logf("proof: %s", enc(proof[:]))

	hash1, err2 := ProofToHash(proof)
	if err2 != nil {
		t.Fatalf("failed to hash: %s", err2)
	}
	t.Logf("hash for \"%s\": %s", message, hash1.ToInt())

	verified, err3 := Verify(publicKey, proof, message)
	if err3 != nil {
		t.Errorf("failed to verify: %s", err3)
	} else if !verified {
		t.Errorf("incompatible output")
	}
}

func TestAvalancheEffect(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.NewKeyFromSeed(secret[:])

	for _, messageString := range Message {
		message := []byte(messageString)

		proof, err := Prove(privateKey, message)
		require.NoError(t, err)
		hash, err := ProofToHash(proof)
		require.NoError(t, err)

		var avalanche []float32
		n := len(message) * 8
		avalanche = make([]float32, n)

		for i := 0; i < n; i++ {
			old := message[i/8]
			message[i/8] ^= byte(uint(1) << (uint(i) % uint(8))) // modify 1 bit

			proof2, err := Prove(privateKey, message)
			require.NoError(t, err)
			hash2, err := ProofToHash(proof2)
			require.NoError(t, err)

			avalanche[i] = getAvalanche(hash, hash2)

			// restore old value
			message[i/8] = old
		}

		var result string
		for j := 0; j < n; j++ {
			result = fmt.Sprintf("%s, %.2f", result, avalanche[j])
		}
		t.Logf(result)
	}
}

func getAvalanche(a []byte, b []byte) (avalanche float32) {
	var count int
	for i := 0; i < len(a); i++ {
		for j := 0; j < 8; j++ {
			if (a[i] & byte(uint(1)<<uint(j))) == (b[i] & byte(uint(1)<<uint(j))) {
				count++
			}
		}
	}
	avalanche = float32(count) / float32(len(a)*8)
	return
}
