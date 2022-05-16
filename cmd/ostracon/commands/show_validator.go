package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/line/ostracon/crypto"
	tmjson "github.com/line/ostracon/libs/json"
	tmos "github.com/line/ostracon/libs/os"
	"github.com/line/ostracon/node"
	"github.com/line/ostracon/privval"
)

// ShowValidatorCmd adds capabilities for showing the validator info.
var ShowValidatorCmd = &cobra.Command{
	Use:     "show-validator",
	Aliases: []string{"show_validator"},
	Short:   "Show this node's validator info",
	RunE:    showValidator,
	PreRun:  deprecateSnakeCase,
}

func showValidator(cmd *cobra.Command, args []string) error {
	var pubKey crypto.PubKey
	var err error
	if config.PrivValidatorListenAddr != "" {
		chainID := "" // currently not in use
		pubKey, err = node.ObtainRemoteSignerPubKeyInformally(config.PrivValidatorListenAddr, chainID, logger)
	} else {
		keyFilePath := config.PrivValidatorKeyFile()
		if !tmos.FileExists(keyFilePath) {
			return fmt.Errorf("private validator file %s does not exist", keyFilePath)
		}
		pv := privval.LoadFilePV(keyFilePath, config.PrivValidatorStateFile())
		pubKey, err = pv.GetPubKey()
	}

	if err != nil {
		return fmt.Errorf("can't get pubkey: %w", err)
	}

	bz, err := tmjson.Marshal(pubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private validator pubkey: %w", err)
	}

	fmt.Println(string(bz))
	return nil
}
