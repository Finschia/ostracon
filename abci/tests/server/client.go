package testsuite

import (
	"bytes"
	"errors"
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/line/ostracon/abci/client"
	"github.com/line/ostracon/abci/example/kvstore"
	ocabci "github.com/line/ostracon/abci/types"
	tmrand "github.com/line/ostracon/libs/rand"
)

func InitChain(client abcicli.Client) error {
	total := 10
	vals := make([]abci.ValidatorUpdate, total)
	for i := 0; i < total; i++ {
		pubkey := kvstore.GenDefaultPrivKey().PubKey()
		power := tmrand.Int()
		vals[i] = ocabci.NewValidatorUpdate(pubkey, int64(power))
	}
	_, err := client.InitChainSync(abci.RequestInitChain{
		Validators: vals,
	})
	if err != nil {
		fmt.Printf("Failed test: InitChain - %v\n", err)
		return err
	}
	fmt.Println("Passed test: InitChain")
	return nil
}

func SetOption(client abcicli.Client, key, value string) error {
	_, err := client.SetOptionSync(abci.RequestSetOption{Key: key, Value: value})
	if err != nil {
		fmt.Println("Failed test: SetOption")
		fmt.Printf("error while setting %v=%v: \nerror: %v\n", key, value, err)
		return err
	}
	fmt.Println("Passed test: SetOption")
	return nil
}

func Commit(client abcicli.Client, hashExp []byte) error {
	res, err := client.CommitSync()
	data := res.Data
	if err != nil {
		fmt.Println("Failed test: Commit")
		fmt.Printf("error while committing: %v\n", err)
		return err
	}
	if !bytes.Equal(data, hashExp) {
		fmt.Println("Failed test: Commit")
		fmt.Printf("Commit hash was unexpected. Got %X expected %X\n", data, hashExp)
		return errors.New("commitTx failed")
	}
	fmt.Println("Passed test: Commit")
	return nil
}

func DeliverTx(client abcicli.Client, txBytes []byte, codeExp uint32, dataExp []byte) error {
	res, _ := client.DeliverTxSync(abci.RequestDeliverTx{Tx: txBytes})
	code, data, log := res.Code, res.Data, res.Log
	if code != codeExp {
		fmt.Println("Failed test: DeliverTx")
		fmt.Printf("DeliverTx response code was unexpected. Got %v expected %v. Log: %v\n",
			code, codeExp, log)
		return errors.New("deliverTx error")
	}
	if !bytes.Equal(data, dataExp) {
		fmt.Println("Failed test: DeliverTx")
		fmt.Printf("DeliverTx response data was unexpected. Got %X expected %X\n",
			data, dataExp)
		return errors.New("deliverTx error")
	}
	fmt.Println("Passed test: DeliverTx")
	return nil
}

func CheckTx(client abcicli.Client, txBytes []byte, codeExp uint32, dataExp []byte) error {
	res, _ := client.CheckTxSync(abci.RequestCheckTx{Tx: txBytes})
	code, data, log := res.Code, res.Data, res.Log
	if code != codeExp {
		fmt.Println("Failed test: CheckTx")
		fmt.Printf("CheckTx response code was unexpected. Got %v expected %v. Log: %v\n",
			code, codeExp, log)
		return errors.New("checkTx")
	}
	if !bytes.Equal(data, dataExp) {
		fmt.Println("Failed test: CheckTx")
		fmt.Printf("CheckTx response data was unexpected. Got %X expected %X\n",
			data, dataExp)
		return errors.New("checkTx")
	}
	fmt.Println("Passed test: CheckTx")
	return nil
}
