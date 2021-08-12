# VRF

VRF implementation is set by `func init()` with `build` option

## Interface
* package/file
  * line/ostracon/crypto/vrf
    * `var defaultVrf vrfEd25519`
    * vrf.go
    * vrf_test.go
```go
type vrfEd25519 interface {
	Prove(privateKey []byte, message []byte) (Proof, error)
	Verify(publicKey []byte, proof Proof, message []byte) (bool, error)
	ProofToHash(proof Proof) (Output, error)
```

## Implementations

Use `func init()` with `build` option

* package/file
  * line/ostracon/crypto/vrf
    * (r2ishiguro = default)
      * `// +build !libsodium,!coniks`
      * `func init() { defaultVrf = newVrfEd25519r2ishiguro() }`
      * vrf_r2ishiguro.go
    * (coniks)
      * `// +build coniks`
      * `func init() { defaultVrf = newVrfEd25519coniks() }`
      * vrf_coniks.go
      * vrf_coniks_test.go
    * (libsodium)
      * `// +build libsodium`
      * `func init() { defaultVrf = newVrfEd25519libsodium() }`
      * vrf_libsodium.go
      * vrf_libsodium_test.go

### Status

| impl | available | memo |
|:---|:---|:---|
|r2ishiguro|o|(default)|
|coniks|x|no compatibility between *crypto ED25519* and *coniks ED25519* (See `TestProveAndVerify_ConiksByCryptoED25519`)|
|libsodium|o| need to build libsodium (See `libsodium` task of `Makefile`)|

### Attention

* There is no compatibility between *r2ishiguro.Prove/libsodium.Verify* and *libsodium.Prove/r2ishiguro.Verify* (See `TestProveAndVerifyCompatibilityLibsodium`)
* Ostracon Network should use `r2ishiguro` or `libsodium` (Can't use both at the same time in Ostracon Network)

### libsodium (bind C implementations)
* package/file
  * line/ostracon/crypto/vrf/internal/vrf
    * `// +build libsodium`
    * vrf.go
    * vrf_test.go
    * libsodium: submodule (See `.gitmodule`)
    * sodium: libs (See `libsodium` task of `Makefile`)

## How to test

```shell
# r2ishiguro
go test github.com/line/ostracon/crypto/vrf -tags r2ishiguro
# libsodium
go test github.com/line/ostracon/crypto/vrf -tags libsodium
# internal libsodium only
go test github.com/line/ostracon/crypto/vrf/internal/vrf -v -tags libsodium

# coniks is not available, but if you want to do, you can see no-compatibility
go test github.com/line/ostracon/crypto/vrf -tags coniks
```

## How to benchmark

```shell
# r2ishiguro
go test -bench Benchmark github.com/line/ostracon/crypto/vrf -run ^$ -benchtime=1000x -count 10 -benchmem -v
# libsodium
go test -bench Benchmark github.com/line/ostracon/crypto/vrf -run ^$ -benchtime=1000x -count 10 -benchmem -v -tags libsodium
```

## How to build

```shell
# r2ishiguro
make build
# libsodium
LIBSODIUM=1 make build
```
