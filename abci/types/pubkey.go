package types

import (
	"github.com/tendermint/tendermint/crypto"
	cryptoenc "github.com/tendermint/tendermint/crypto/encoding"
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
