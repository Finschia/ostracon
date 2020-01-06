package vrf_test

import (
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "github.com/tendermint/tendermint/crypto/ed25519"
    "github.com/tendermint/tendermint/crypto/vrf"
    vrfimpl "github.com/tendermint/tendermint/crypto/vrf/internal/vrf"
    "math/rand"
    "testing"
    "unsafe"
)

func enc(s []byte) string {
    return hex.EncodeToString(s)
}

func TestConstants(t *testing.T) {
    t.Logf("PUBLICKEYBYTES: %d\n", vrfimpl.PUBLICKEYBYTES)
    t.Logf("SECRETKEYBYTES: %d\n", vrfimpl.SECRETKEYBYTES)
    t.Logf("SEEDBYTES: %d\n", vrfimpl.SEEDBYTES)
    t.Logf("PROOFBYTES: %d\n", vrfimpl.PROOFBYTES)
    t.Logf("OUTPUTBYTES: %d\n", vrfimpl.OUTPUTBYTES)
    t.Logf("PRIMITIVE: %s\n", vrfimpl.PRIMITIVE)

    if vrfimpl.PUBLICKEYBYTES != 32 {
        t.Errorf("public key size: %d != 32\n", vrfimpl.PUBLICKEYBYTES)
    }
    if vrfimpl.SECRETKEYBYTES != 64 {
        t.Errorf("secret key size: %d != 64\n", vrfimpl.SECRETKEYBYTES)
    }
    if vrfimpl.SEEDBYTES != 32 {
        t.Errorf("seed size: %d != 32\n", vrfimpl.SEEDBYTES)
    }
    if vrfimpl.OUTPUTBYTES != 64 {
        t.Errorf("output size: %d != 64\n", vrfimpl.OUTPUTBYTES)
    }
    if vrfimpl.PRIMITIVE != "ietfdraft03" {
        t.Errorf("primitive: %s != \"ietfdraft03\"\n", vrfimpl.PRIMITIVE)
    }
}

func TestKeyPair(t *testing.T) {
    var pk, sk = vrfimpl.KeyPair()
    t.Logf("random public key: %s (%d bytes)\n", enc(pk[:]), len(pk))
    t.Logf("random private key: %s (%d bytes)\n", enc(sk[:]), len(sk))
    if uint32(len(pk)) != vrfimpl.PUBLICKEYBYTES {
        t.Errorf("public key size: %d != %d", len(pk), vrfimpl.PUBLICKEYBYTES)
    }
    if uint32(len(sk)) != vrfimpl.SECRETKEYBYTES {
        t.Errorf("secret key size: %d != %d", len(sk), vrfimpl.SECRETKEYBYTES)
    }
}

func TestKeyPairFromSeed(t *testing.T) {
    var seed [vrfimpl.SEEDBYTES]byte
    var pk, sk = vrfimpl.KeyPairFromSeed(&seed)
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
    if uint32(len(pk)) != vrfimpl.PUBLICKEYBYTES {
        t.Errorf("public key size: %d != %d", len(pk), vrfimpl.PUBLICKEYBYTES)
    }
    if uint32(len(sk)) != vrfimpl.SECRETKEYBYTES {
        t.Errorf("secret key size: %d != %d", len(sk), vrfimpl.SECRETKEYBYTES)
    }

    var message [0]byte
    var proof, err1 = vrfimpl.Prove(sk, message[:])
    if err1 != nil {
        t.Errorf("probe failed: %s", err1)
    }
    t.Logf("proof: %s (%d bytes)\n", enc(proof[:]), len(proof))
    var output, err2 = vrfimpl.ProofToHash(proof)
    if err2 != nil {
        t.Errorf("failed to hash proof: %s", err2)
    }
    t.Logf("output: %s (%d bytes)\n", enc(output[:]), len(output))
}

func TestIsValidKey(t *testing.T) {

    // generated from KeyPair()
    var pk1, _ = vrfimpl.KeyPair()
    if ! vrfimpl.IsValidKey(pk1) {
        t.Errorf("public key is not valid: %s", enc(pk1[:]))
    }

    // generated from KeyPairFromSeed()
    var seed [vrfimpl.SEEDBYTES]byte
    var pk2, _ = vrfimpl.KeyPairFromSeed(&seed)
    if ! vrfimpl.IsValidKey(pk2) {
        t.Errorf("public key is not valid: %s", enc(pk2[:]))
    }

    // zero
    var zero [vrfimpl.PUBLICKEYBYTES]byte
    if vrfimpl.IsValidKey(&zero) {
        t.Error("recognized as valid for zero pk")
    }

    // random bytes
    var random [vrfimpl.PUBLICKEYBYTES]byte
    var rng = rand.New(rand.NewSource(0))
    rng.Read(random[:])
    if vrfimpl.IsValidKey(&random) {
        t.Errorf("recognized as valid for random pk: %s", enc(random[:]))
    }
}

func TestProveAndVerify(t *testing.T) {
    message := []byte("hello, world")

    var zero [vrfimpl.SEEDBYTES]byte
    var pk, sk = vrfimpl.KeyPairFromSeed(&zero)
    var proof, err1 = vrfimpl.Prove(sk, message)
    if err1 != nil {
        t.Errorf("probe failed: %s", err1)
    }
    var output, err2 = vrfimpl.ProofToHash(proof)
    if err2 != nil {
        t.Errorf("failed to hash proof: %s", err2)
    }
    t.Logf("SEED[%s] -> OUTPUT[%s]\n", enc(zero[:]), enc(output[:]))
    var expected, err3 = vrfimpl.Verify(pk, proof, message)
    if err3 != nil {
        t.Errorf("validation failed: %s", err3)
    } else if bytes.Compare(expected[:], output[:]) != 0 {
        t.Errorf("output not matches: %s", enc(output[:]))
    }

    // essentially, the private key for ed25519 could be any value at a point on the finite field.
    var invalidPrivateKey [vrfimpl.SECRETKEYBYTES]byte
    for i := range invalidPrivateKey {
        invalidPrivateKey[i] = 0xFF
    }
    var _, err4 = vrfimpl.Prove(&invalidPrivateKey, message)
    if err4 == nil {
        t.Errorf("Prove() with invalid private key didn't fail")
    }

    // unexpected public key for Verify()
    var zero3 [vrfimpl.PUBLICKEYBYTES]byte
    var _, err5 = vrfimpl.Verify(&zero3, proof, message)
    if err5 == nil {
        t.Errorf("Verify() with zero public key didn't fail")
    }

    // unexpected proof for Verify()
    var zero4 [vrfimpl.PROOFBYTES]byte
    var _, err6 = vrfimpl.Verify(pk, &zero4, message)
    if err6 == nil {
        t.Errorf("Verify() with zero proof didn't fail")
    }

    // unexpected message for Verify()
    var message2 = []byte("good-by world")
    var output2, err7 = vrfimpl.Verify(pk, proof, message2)
    if err7 == nil {
        t.Errorf("Verify() success without error: %s", enc(output2[:]))
    }

    // essentially, the proof for ed25519 could be any value at a point on the finite field.
    var invalidProof [vrfimpl.PROOFBYTES]byte
    for i := range invalidProof {
        invalidProof[i] = 0xFF
    }
    var _, err8 = vrfimpl.ProofToHash(&invalidProof)
    if err8 == nil {
        t.Errorf("ProofToHash() with invalid proof didn't fail")
    }
}

func TestSkToPk(t *testing.T) {
    var zero [vrfimpl.SEEDBYTES]byte
    var expected, sk = vrfimpl.KeyPairFromSeed(&zero)

    var actual = vrfimpl.SkToPk(sk)

    if bytes.Compare(expected[:], actual[:]) != 0 {
        t.Errorf("public key didn't match: %s != %s", enc(expected[:]), enc(actual[:]))
    }
}

func TestSkToSeed(t *testing.T) {
    var zero [vrfimpl.SEEDBYTES]byte
    var _, sk = vrfimpl.KeyPairFromSeed(&zero)

    var actual = vrfimpl.SkToSeed(sk)

    if bytes.Compare(zero[:], actual[:]) != 0 {
        t.Errorf("seed didn't match: %s != %s", enc(zero[:]), enc(actual[:]))
    }
}

func TestToHash(t *testing.T) {
    secret := [vrfimpl.SEEDBYTES]byte{}
    privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
    message := []byte("hello, world")

    proof, err1 := vrf.Prove(&privateKey, message)
    if err1 != nil {
        t.Fatalf("failed to prove: %s", err1)
    }

    _, err2 := proof.ToHash()
    if err2 != nil {
        t.Errorf("failed to convert to hash: %s", enc(proof[:]))
    }

    // check to fail for invalid proof bytes
    for i := range proof {
        proof[i] = 0xFF
    }
    op3, err3 := proof.ToHash()
    if err3 == nil {
        t.Errorf("unexpected hash for invalid proof: %s", enc(op3[:]))
    }
}

func TestKeyPairCompatibility(t *testing.T) {
    var secret [vrfimpl.SEEDBYTES]byte
    tmPrivKey := ed25519.GenPrivKeyFromSecret(secret[:])
    tmPubKey, _ := tmPrivKey.PubKey().(ed25519.PubKeyEd25519)
    tmPrivKeyBytes := tmPrivKey[:]
    tmPubKeyBytes := tmPubKey[:]

    var seed [vrfimpl.SEEDBYTES]byte
    hashedSecret := sha256.Sum256(secret[:])
    copy(seed[:], hashedSecret[:])
    lsPubKey, lsPrivKey := vrfimpl.KeyPairFromSeed(&seed)

    if ! bytes.Equal(tmPrivKeyBytes, lsPrivKey[:]) {
        t.Errorf("incompatible private key: %s != %s",
            enc(tmPrivKeyBytes), enc(lsPrivKey[:]))
    }
    t.Logf("tendermint: private key: %s (%d bytes)\n", enc(tmPrivKeyBytes[:]), len(tmPrivKey))
    t.Logf("libsodium : private key: %s (%d bytes)\n", enc(lsPrivKey[:]), len(lsPrivKey))

    if ! bytes.Equal(tmPubKeyBytes, lsPubKey[:]) {
        t.Errorf("incompatible public key: %s != %s", enc(tmPubKeyBytes), enc(lsPubKey[:]))
    }
    t.Logf("tendermint: public key: %s (%d bytes)\n", enc(tmPubKeyBytes), len(tmPubKey))
    t.Logf("libsodium : public key: %s (%d bytes)\n", enc(lsPubKey[:]), len(lsPubKey))

    pubKeyBytesPtr := (*[vrfimpl.PUBLICKEYBYTES]byte)(unsafe.Pointer(&tmPubKey))
    if ! vrfimpl.IsValidKey(pubKeyBytesPtr) {
        t.Errorf("ed25519 key is not a valid public key")
    }
}

func TestProve(t *testing.T) {
    secret := [vrfimpl.SEEDBYTES]byte{}
    privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
    publicKey, _ := privateKey.PubKey().(ed25519.PubKeyEd25519)
    t.Logf("seed: %s", enc(secret[:]))
    t.Logf("private key: [%s]", enc(privateKey[:]))
    t.Logf("public  key: [%s]", enc(publicKey[:]))

    message := []byte("hello, world")
    proof, err1 := vrf.Prove(&privateKey, message)
    if err1 != nil {
        t.Fatalf("failed to prove: %s", err1)
    }
    t.Logf("proof: %s", enc(proof[:]))

    hash1, err2 := proof.ToHash()
    if err2 != nil {
        t.Fatalf("failed to hash: %s", err2)
    }
    t.Logf("hash for \"%s\": %s", message, hash1.ToInt())

    hash2, err3 := vrf.Verify(&publicKey, proof, message)
    if err3 != nil {
        t.Errorf("failed to verify: %s", err3)
    } else if ! bytes.Equal(hash1[:], hash2[:]) {
        t.Errorf("incompatible output: %s != %s", enc(hash1[:]), enc(hash2[:]))
    }
}
