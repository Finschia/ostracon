package rand

import (
	crand "crypto/rand"
)

func Read(b []byte) {
	if _, err := crand.Read(b); err != nil {
		panic(err)
	}
}
