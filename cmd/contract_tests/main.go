package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Finschia/ostracon/cmd/contract_tests/unmarshaler"

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

	// TODO This check need to remove if dredd is updated to check optional
	// dredd can not validate optional items
	h.Before("/genesis > Get Genesis > 200 > application/json", func(t *transaction.Transaction) {
		removeOptionalFieldsOfExpected(t, []string{"result.genesis.app_state"})
	})
	h.Before("/broadcast_tx_async > Returns right away, with no response. "+
		"Does not wait for CheckTx nor DeliverTx results. > 200 > application/json", func(t *transaction.Transaction) {
		removeOptionalFieldsOfExpected(t, []string{"error"})
	})
	h.Before("/broadcast_tx_sync > Returns with the response from CheckTx. "+
		"Does not wait for DeliverTx result. > 200 > application/json", func(t *transaction.Transaction) {
		removeOptionalFieldsOfExpected(t, []string{"error"})
	})
	h.Before("/broadcast_tx_commit > Returns with the responses from CheckTx and DeliverTx. "+
		"> 200 > application/json", func(t *transaction.Transaction) {
		removeOptionalFieldsOfExpected(t, []string{"error"})
	})
	h.Before("/block_results > Get block results at a specified height > 200 > application/json",
		func(t *transaction.Transaction) {
			removeOptionalFieldsOfExpected(t, []string{
				"result.txs_results",
				"result.begin_block_events",
				"result.end_block",
				"result.end_block_events",
				"result.validator_updates",
				"result.consensus_param_updates"})
		})

	server.Serve()
	defer server.Listener.Close()
}

func removeOptionalFieldsOfExpected(t *transaction.Transaction, paths []string) {
	expected := unmarshaler.UnmarshalJSON(&t.Expected.Body)
	for _, path := range paths {
		expected.DeleteProperty(strings.Split(path, ".")...)
	}
	newBody, err := json.Marshal(expected.Body)
	if err != nil {
		panic(fmt.Sprintf("fail to marshal expected body with %s", err))
	}
	t.Expected.Body = string(newBody)
}
