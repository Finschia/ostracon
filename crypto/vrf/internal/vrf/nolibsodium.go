// +build !libsodium

package vrf

// currently this constants are not used
// but it is necessary to avoid 'no Go source files' error and
// red compilation error lines in IDE if we don't put libsodium build tag at build command
const (
	PUBLICKEYBYTES = 0
	SECRETKEYBYTES = 0
	SEEDBYTES      = 0
	PROOFBYTES     = 0
	OUTPUTBYTES    = 0
	PRIMITIVE      = 0
)
