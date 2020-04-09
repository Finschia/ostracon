package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tendermint/tendermint/cmd/contract_tests/unmarshaler"

	"github.com/snikch/goodman/hooks"
	"github.com/snikch/goodman/transaction"
)

func main() {
	// This must be compiled beforehand and given to dredd as parameter, in the meantime the server should be running
	h := hooks.NewHooks()
	server := hooks.NewServer(hooks.NewHooksRunner(h))
	h.BeforeAll(func(t []*transaction.Transaction) {
		fmt.Println(t[0].Name)
	})
	h.BeforeEach(func(t *transaction.Transaction) {
		if t.Expected.StatusCode != "200" {
			t.Skip = true
		} else if strings.HasPrefix(t.Name, "Tx") ||
			// We need a proper example of evidence to broadcast
			strings.HasPrefix(t.Name, "/broadcast_evidence >") ||
			// We need a proper example of path and data
			strings.HasPrefix(t.Name, "/abci_query >") ||
			// We need to find a way to make a transaction before starting the tests,
			// that hash should replace the dummy one in the openapi file
			strings.HasPrefix(t.Name, "/tx >") {
			t.Skip = true
		}
	})

	// dredd can not validate optional items
	h.Before("/genesis > Get Genesis > 200 > application/json", func(t *transaction.Transaction) {
		makeExpectedGenesis(t)
	})
	server.Serve()
	defer server.Listener.Close()
}

// add optional field of genesis response, because dredd has bug about OA3
func makeExpectedGenesis(t *transaction.Transaction) {
	expected := unmarshaler.UnmarshalJSON(&t.Expected.Body)
	expected.DeleteProperty("result", "genesis", "app_state")
	newBody, err := json.Marshal(expected.Body)
	if err != nil {
		panic(fmt.Sprintf("fail to marshal expected body with %s", err))
	}
	t.Expected.Body = string(newBody)
}
