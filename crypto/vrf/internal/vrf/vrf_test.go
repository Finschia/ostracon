// +build libsodium

package vrf

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"testing"
	"unsafe"

	"github.com/line/ostracon/crypto/ed25519"
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
	t.Logf("PUBLICKEYBYTES: %d\n", PUBLICKEYBYTES)
	t.Logf("SECRETKEYBYTES: %d\n", SECRETKEYBYTES)
	t.Logf("SEEDBYTES: %d\n", SEEDBYTES)
	t.Logf("PROOFBYTES: %d\n", PROOFBYTES)
	t.Logf("OUTPUTBYTES: %d\n", OUTPUTBYTES)
	t.Logf("PRIMITIVE: %s\n", PRIMITIVE)

	if PUBLICKEYBYTES != 32 {
		t.Errorf("public key size: %d != 32\n", PUBLICKEYBYTES)
	}
	if SECRETKEYBYTES != 64 {
		t.Errorf("secret key size: %d != 64\n", SECRETKEYBYTES)
	}
	if SEEDBYTES != 32 {
		t.Errorf("seed size: %d != 32\n", SEEDBYTES)
	}
	if OUTPUTBYTES != 64 {
		t.Errorf("output size: %d != 64\n", OUTPUTBYTES)
	}
	if PRIMITIVE != "ietfdraft03" {
		t.Errorf("primitive: %s != \"ietfdraft03\"\n", PRIMITIVE)
	}
}

func TestKeyPair(t *testing.T) {
	var pk, sk = KeyPair()
	t.Logf("random public key: %s (%d bytes)\n", enc(pk[:]), len(pk))
	t.Logf("random private key: %s (%d bytes)\n", enc(sk[:]), len(sk))
	if uint32(len(pk)) != PUBLICKEYBYTES {
		t.Errorf("public key size: %d != %d", len(pk), PUBLICKEYBYTES)
	}
	if uint32(len(sk)) != SECRETKEYBYTES {
		t.Errorf("secret key size: %d != %d", len(sk), SECRETKEYBYTES)
	}
}

func TestKeyPairFromSeed(t *testing.T) {
	var seed [SEEDBYTES]byte
	var pk, sk = KeyPairFromSeed(&seed)
	t.Logf("static seed: %s (%d bytes)\n", enc(seed[:]), len(seed))
	t.Logf("static public key: %s (%d bytes)\n", enc(pk[:]), len(pk))
	t.Logf("static private key: %s (%d bytes)\n", enc(sk[:]), len(sk))
	if enc(pk[:]) != "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29" {
		t.Errorf("unexpected public key: %s", enc(pk[:]))
	}
	if enc(sk[:]) != "0000000000000000000000000000000000000000000000000000000000000000"+
		"3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29" {
		t.Errorf("unexpected private key: %s", enc(sk[:]))
	}
	if uint32(len(pk)) != PUBLICKEYBYTES {
		t.Errorf("public key size: %d != %d", len(pk), PUBLICKEYBYTES)
	}
	if uint32(len(sk)) != SECRETKEYBYTES {
		t.Errorf("secret key size: %d != %d", len(sk), SECRETKEYBYTES)
	}

	var message [0]byte
	var proof, err1 = Prove(sk, message[:])
	if err1 != nil {
		t.Errorf("probe failed: %s", err1)
	}
	t.Logf("proof: %s (%d bytes)\n", enc(proof[:]), len(proof))
	var output, err2 = ProofToHash(proof)
	if err2 != nil {
		t.Errorf("failed to hash proof: %s", err2)
	}
	t.Logf("output: %s (%d bytes)\n", enc(output[:]), len(output))
}

func TestHashIsDeterministicForKeyPairAndMessage(t *testing.T) {
	sk := ed25519.GenPrivKey()
	pk, _ := sk.PubKey().(ed25519.PubKeyEd25519)
	message := []byte("hello, world")
	var hashes = []*Output{}
	var proofs = []*Proof{}
	for i := 0; i < 100; i++ {
		var proof, err1 = prove(&sk, message[:])
		if err1 != nil {
			t.Errorf("probe failed: %s", err1)
		} else {
			hash, err2 := proof.ToHash()
			if err2 != nil {
				t.Errorf("failed to hash proof: %s", err2)
			} else {
				output, err3 := verify(&pk, proof, message)
				if err3 != nil {
					t.Errorf("fail to verify proof: %s", err3)
				} else if !bytes.Equal(hash[:], output[:]) {
					t.Errorf("hash not match")
				} else {
					hashes = append(hashes, hash)
					proofs = append(proofs, proof)
				}
			}
		}
	}

	t.Logf("proofs for \"%s\": %s × %d", string(message), hex.EncodeToString(proofs[0][:]), len(hashes))
	t.Logf("hashes for \"%s\": %s × %d", string(message), hex.EncodeToString(hashes[0][:]), len(hashes))

	hash := hashes[0]
	proof := proofs[0]
	for i := 1; i < len(hashes); i++ {
		if !bytes.Equal(hash[:], hashes[i][:]) {
			t.Errorf("contains different hash: %s != %s",
				hex.EncodeToString(hash[:]), hex.EncodeToString(hashes[i][:]))
		}
		if !bytes.Equal(proof[:], proofs[i][:]) {
			t.Errorf("contains different proof: %s != %s",
				hex.EncodeToString(proof[:]), hex.EncodeToString(proofs[i][:]))
		}
	}
}

func TestIsValidKey(t *testing.T) {

	// generated from KeyPair()
	var pk1, _ = KeyPair()
	if !IsValidKey(pk1) {
		t.Errorf("public key is not valid: %s", enc(pk1[:]))
	}

	// generated from KeyPairFromSeed()
	var seed [SEEDBYTES]byte
	var pk2, _ = KeyPairFromSeed(&seed)
	if !IsValidKey(pk2) {
		t.Errorf("public key is not valid: %s", enc(pk2[:]))
	}

	// zero
	var zero [PUBLICKEYBYTES]byte
	if IsValidKey(&zero) {
		t.Error("recognized as valid for zero pk")
	}

	// random bytes
	var random [PUBLICKEYBYTES]byte
	var rng = rand.New(rand.NewSource(0))
	rng.Read(random[:])
	if IsValidKey(&random) {
		t.Errorf("recognized as valid for random pk: %s", enc(random[:]))
	}
}

func TestProveAndVerify(t *testing.T) {
	message := []byte("hello, world")

	var zero [SEEDBYTES]byte
	var pk, sk = KeyPairFromSeed(&zero)
	var proof, err1 = Prove(sk, message)
	if err1 != nil {
		t.Errorf("probe failed: %s", err1)
	}
	var output, err2 = ProofToHash(proof)
	if err2 != nil {
		t.Errorf("failed to hash proof: %s", err2)
	}
	t.Logf("SEED[%s] -> OUTPUT[%s]\n", enc(zero[:]), enc(output[:]))
	var expected, err3 = Verify(pk, proof, message)
	if err3 != nil {
		t.Errorf("validation failed: %s", err3)
	} else if bytes.Compare(expected[:], output[:]) != 0 {
		t.Errorf("output not matches: %s", enc(output[:]))
	}

	// essentially, the private key for ed25519 could be any value at a point on the finite field.
	var invalidPrivateKey [SECRETKEYBYTES]byte
	for i := range invalidPrivateKey {
		invalidPrivateKey[i] = 0xFF
	}
	var _, err4 = Prove(&invalidPrivateKey, message)
	if err4 == nil {
		t.Errorf("Prove() with invalid private key didn't fail")
	}

	// unexpected public key for Verify()
	var zero3 [PUBLICKEYBYTES]byte
	var _, err5 = Verify(&zero3, proof, message)
	if err5 == nil {
		t.Errorf("Verify() with zero public key didn't fail")
	}

	// unexpected proof for Verify()
	var zero4 [PROOFBYTES]byte
	var _, err6 = Verify(pk, &zero4, message)
	if err6 == nil {
		t.Errorf("Verify() with zero proof didn't fail")
	}

	// unexpected message for Verify()
	var message2 = []byte("good-by world")
	var output2, err7 = Verify(pk, proof, message2)
	if err7 == nil {
		t.Errorf("Verify() success without error: %s", enc(output2[:]))
	}

	// essentially, the proof for ed25519 could be any value at a point on the finite field.
	var invalidProof [PROOFBYTES]byte
	for i := range invalidProof {
		invalidProof[i] = 0xFF
	}
	var _, err8 = ProofToHash(&invalidProof)
	if err8 == nil {
		t.Errorf("ProofToHash() with invalid proof didn't fail")
	}
}

func TestSkToPk(t *testing.T) {
	var zero [SEEDBYTES]byte
	var expected, sk = KeyPairFromSeed(&zero)

	var actual = SkToPk(sk)

	if bytes.Compare(expected[:], actual[:]) != 0 {
		t.Errorf("public key didn't match: %s != %s", enc(expected[:]), enc(actual[:]))
	}
}

func TestSkToSeed(t *testing.T) {
	var zero [SEEDBYTES]byte
	var _, sk = KeyPairFromSeed(&zero)

	var actual = SkToSeed(sk)

	if bytes.Compare(zero[:], actual[:]) != 0 {
		t.Errorf("seed didn't match: %s != %s", enc(zero[:]), enc(actual[:]))
	}
}

func TestKeyPairCompatibility(t *testing.T) {
	var secret [SEEDBYTES]byte
	tmPrivKey := ed25519.GenPrivKeyFromSecret(secret[:])
	tmPubKey, _ := tmPrivKey.PubKey().(ed25519.PubKeyEd25519)
	tmPrivKeyBytes := tmPrivKey[:]
	tmPubKeyBytes := tmPubKey[:]

	var seed [SEEDBYTES]byte
	hashedSecret := sha256.Sum256(secret[:])
	copy(seed[:], hashedSecret[:])
	lsPubKey, lsPrivKey := KeyPairFromSeed(&seed)

	if !bytes.Equal(tmPrivKeyBytes, lsPrivKey[:]) {
		t.Errorf("incompatible private key: %s != %s",
			enc(tmPrivKeyBytes), enc(lsPrivKey[:]))
	}
	t.Logf("ostracon: private key: %s (%d bytes)\n", enc(tmPrivKeyBytes[:]), len(tmPrivKey))
	t.Logf("libsodium : private key: %s (%d bytes)\n", enc(lsPrivKey[:]), len(lsPrivKey))

	if !bytes.Equal(tmPubKeyBytes, lsPubKey[:]) {
		t.Errorf("incompatible public key: %s != %s", enc(tmPubKeyBytes), enc(lsPubKey[:]))
	}
	t.Logf("ostracon: public key: %s (%d bytes)\n", enc(tmPubKeyBytes), len(tmPubKey))
	t.Logf("libsodium : public key: %s (%d bytes)\n", enc(lsPubKey[:]), len(lsPubKey))

	pubKeyBytesPtr := (*[PUBLICKEYBYTES]byte)(unsafe.Pointer(&tmPubKey))
	if !IsValidKey(pubKeyBytesPtr) {
		t.Errorf("ed25519 key is not a valid public key")
	}

	// random Tendermint's key-pairs
	msg := []byte("hello, world")
	for i := 0; i < 100; i++ {
		privKey := ed25519.GenPrivKey()
		proof, err := prove(&privKey, msg)
		if err != nil {
			t.Errorf("Prove() failed: %s", err)
		} else {
			pubKey, _ := privKey.PubKey().(ed25519.PubKeyEd25519)
			output, err := verify(&pubKey, proof, msg)
			if err != nil {
				t.Errorf("Verify() failed: %s", err)
			} else {
				hash, err := proof.ToHash()
				if err != nil {
					t.Errorf("Proof.ToHash() failed: %s", err)
				} else if !bytes.Equal(hash[:], output[:]) {
					t.Errorf("proof hash and verify hash didn't match: %s != %s",
						hex.EncodeToString(hash[:]), hex.EncodeToString(output[:]))
				}
			}
		}
	}
}

func TestProve(t *testing.T) {
	secret := [SEEDBYTES]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	publicKey, _ := privateKey.PubKey().(ed25519.PubKeyEd25519)
	t.Logf("seed: %s", enc(secret[:]))
	t.Logf("private key: [%s]", enc(privateKey[:]))
	t.Logf("public  key: [%s]", enc(publicKey[:]))

	message := []byte("hello, world")
	proof, err1 := prove(&privateKey, message)
	if err1 != nil {
		t.Fatalf("failed to prove: %s", err1)
	}
	t.Logf("proof: %s", enc(proof[:]))

	hash1, err2 := proof.ToHash()
	if err2 != nil {
		t.Fatalf("failed to hash: %s", err2)
	}
	t.Logf("hash for \"%s\": %s", message, hash1.ToInt())

	hash2, err3 := verify(&publicKey, proof, message)
	if err3 != nil {
		t.Errorf("failed to verify: %s", err3)
	} else if !bytes.Equal(hash1[:], hash2[:]) {
		t.Errorf("incompatible output: %s != %s", enc(hash1[:]), enc(hash2[:]))
	}
}
