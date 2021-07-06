package types

import (
	"github.com/line/ostracon/crypto"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
)

func NewValidatorUpdate(pk crypto.PubKey, power int64) ValidatorUpdate {
	pkp, err := cryptoenc.PubKeyToProto(pk)
	if err != nil {
		panic(err)
	}

	return ValidatorUpdate{
		// Address:
		PubKey: pkp,
		Power:  power,
	}
}
