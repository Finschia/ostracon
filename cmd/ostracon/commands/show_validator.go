package commands

import (
	"fmt"

	"github.com/line/ostracon/node"
	"github.com/line/ostracon/types"
	"github.com/spf13/cobra"

	cfg "github.com/line/ostracon/config"
	tmjson "github.com/line/ostracon/libs/json"
	tmos "github.com/line/ostracon/libs/os"
	"github.com/line/ostracon/privval"
)

// ShowValidatorCmd adds capabilities for showing the validator info.
var ShowValidatorCmd = &cobra.Command{
	Use:     "show-validator",
	Aliases: []string{"show_validator"},
	Short:   "Show this node's validator info",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showValidator(cmd, args, config)
	},
	PreRun: deprecateSnakeCase,
}

func showValidator(cmd *cobra.Command, args []string, config *cfg.Config) error {
	var pv types.PrivValidator
	var err error
	if config.PrivValidatorListenAddr != "" {
		chainID := "" // currently not in use
		pv, err = node.CreateAndStartPrivValidatorSocketClient(config.PrivValidatorListenAddr, chainID, logger)
		if err != nil {
			return err
		}
	} else {
		keyFilePath := config.PrivValidatorKeyFile()
		if !tmos.FileExists(keyFilePath) {
			return fmt.Errorf("private validator file %s does not exist", keyFilePath)
		}
		pv = privval.LoadFilePV(keyFilePath, config.PrivValidatorStateFile())
	}

	pubKey, err := pv.GetPubKey()
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
