package commands

import (
	"fmt"
	"github.com/line/ostracon/node"
	"io"

	"github.com/spf13/cobra"

	cfg "github.com/line/ostracon/config"
	tmos "github.com/line/ostracon/libs/os"
	tmrand "github.com/line/ostracon/libs/rand"
	"github.com/line/ostracon/p2p"
	"github.com/line/ostracon/privval"
	"github.com/line/ostracon/types"
	tmtime "github.com/line/ostracon/types/time"
)

func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Ostracon",
		RunE:  initFiles,
	}

	AddInitFlags(cmd)
	return cmd
}

func AddInitFlags(cmd *cobra.Command) {
	cmd.Flags().String("priv_key_type", config.PrivKeyType,
		"Specify validator's private key type (ed25519 | composite)")

	// priv val flags
	cmd.Flags().String(
		"priv_validator_laddr",
		config.PrivValidatorListenAddr,
		"socket address to listen on for connections from external priv_validator process")
}

func initFiles(cmd *cobra.Command, args []string) error {
	return initFilesWithConfig(config)
}

func initFilesWithConfig(config *cfg.Config) error {
	chainID := fmt.Sprintf("test-chain-%v", tmrand.Str(6))

	// private validator
	var pv types.PrivValidator
	var err error
	if config.PrivValidatorListenAddr != "" {
		// If an address is provided, listen on the socket for a connection from an external signing process.
		pv, err = node.CreateAndStartPrivValidatorSocketClient(config.PrivValidatorListenAddr, chainID, logger)
		if err != nil {
			return fmt.Errorf("error with private validator socket client: %w", err)
		}
	} else {
		privValKeyFile := config.PrivValidatorKeyFile()
		privValStateFile := config.PrivValidatorStateFile()
		privKeyType := config.PrivValidatorKeyType()
		var pv *privval.FilePV
		if tmos.FileExists(privValKeyFile) {
			pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
			logger.Info("Found private validator", "keyFile", privValKeyFile,
				"stateFile", privValStateFile)
		} else {
			pv, err = privval.GenFilePV(privValKeyFile, privValStateFile, privKeyType)
			if err != nil {
				return err
			}
			if pv != nil {
				pv.Save()
			}
			logger.Info("Generated private validator", "keyFile", privValKeyFile,
				"stateFile", privValStateFile)
		}
	}

	defer func() {
		if c, ok := pv.(io.Closer); ok {
			if err := c.Close(); err != nil {
				logger.Debug("Failed to close the socket for remote singer", err)
			}
		}
	}()

	// Save default settings with additional command-line specified options (default settings implicitly saved by
	// root.go will be overwritten).
	config.Save()

	nodeKeyFile := config.NodeKeyFile()
	if tmos.FileExists(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	// genesis file
	genFile := config.GenesisFile()
	if tmos.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		genDoc := types.GenesisDoc{
			ChainID:         chainID,
			GenesisTime:     tmtime.Now(),
			ConsensusParams: types.DefaultConsensusParams(),
			VoterParams:     types.DefaultVoterParams(),
		}
		pubKey, err := pv.GetPubKey()
		if err != nil {
			return fmt.Errorf("can't get pubkey: %w", err)
		}
		genDoc.Validators = []types.GenesisValidator{{
			Address: pubKey.Address(),
			PubKey:  pubKey,
			Power:   10,
		}}

		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	return nil
}
