package kvstore

import (
	"fmt"
	"io/ioutil"

	"github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/composite"
	tmjson "github.com/line/ostracon/libs/json"
	tmos "github.com/line/ostracon/libs/os"
	tmrand "github.com/line/ostracon/libs/rand"
	"github.com/line/ostracon/privval"
)

// LoadPrivValidatorKeyFile Load private key for use in an example or test.
func LoadPrivValidatorKeyFile(keyFilePath string) (*privval.FilePVKey, error) {
	if !tmos.FileExists(keyFilePath) {
		return nil, fmt.Errorf("private validator file %s does not exist", keyFilePath)
	}
	keyJSONBytes, _ := ioutil.ReadFile(keyFilePath)
	pvKey := privval.FilePVKey{}
	err := tmjson.Unmarshal(keyJSONBytes, &pvKey)
	if err != nil {
		return nil, fmt.Errorf("error reading PrivValidator key from %v: %v", keyFilePath, err)
	}
	return &pvKey, nil
}

// GenDefaultPrivKey Generates a default private key for use in an example or test.
func GenDefaultPrivKey() crypto.PrivKey {
	return composite.GenPrivKey()
}

// RandVal creates one random validator, with a key derived
// from the input value
func RandVal(i int) types.ValidatorUpdate {
	pk := GenDefaultPrivKey().PubKey()
	power := tmrand.Uint16() + 1
	v := types.NewValidatorUpdate(pk, int64(power))
	return v
}

// RandVals returns a list of cnt validators for initializing
// the application. Note that the keys are deterministically
// derived from the index in the array, while the power is
// random (Change this if not desired)
func RandVals(cnt int) []types.ValidatorUpdate {
	res := make([]types.ValidatorUpdate, cnt)
	for i := 0; i < cnt; i++ {
		res[i] = RandVal(i)
	}
	return res
}

// InitKVStore initializes the kvstore app with some data,
// which allows tests to pass and is fine as long as you
// don't make any tx that modify the validator state
func InitKVStore(app *PersistentKVStoreApplication) {
	app.InitChain(types.RequestInitChain{
		Validators: RandVals(1),
	})
}
