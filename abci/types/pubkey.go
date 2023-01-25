package types

import (
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/line/ostracon/crypto"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
)

func NewValidatorUpdate(pk crypto.PubKey, power int64) abci.ValidatorUpdate {
	pkp, err := cryptoenc.PubKeyToProto(pk)
	if err != nil {
		panic(err)
	}

	return abci.ValidatorUpdate{
		// Address:
		PubKey: pkp,
		Power:  power,
	}
}
