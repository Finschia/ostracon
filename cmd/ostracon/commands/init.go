package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	cfg "github.com/line/ostracon/config"
	"github.com/line/ostracon/crypto"
	tmos "github.com/line/ostracon/libs/os"
	"github.com/line/ostracon/node"
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

func initFilesWithConfig(config *cfg.Config) (err error) {
	chainID := config.ChainID()

	// private validator
	var pubKey crypto.PubKey
	if config.PrivValidatorListenAddr != "" {
		// If an address is provided, listen on the socket for a connection from an external signing process.
		pubKey, err = node.ObtainRemoteSignerPubKeyInformally(config.PrivValidatorListenAddr, chainID, logger)
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
				return
			}
			if pv != nil {
				pv.Save()
			}
			logger.Info("Generated private validator", "keyFile", privValKeyFile,
				"stateFile", privValStateFile)
		}
		pubKey, err = pv.GetPubKey()
		if err != nil {
			return
		}
	}

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

	// Save default settings with additional command-line specified options (default settings implicitly saved by
	// root.go will be overwritten) when all operations succeed.
	// If no configuration file exists when the `init` subcommand is executed and a new file is implicitly
	// created with default settings, overwrite with specified ones by the command line options.
	configFile = config.Path()
	if !configFileGenerated && tmos.FileExists(configFile) {
		logger.Info("Found existing configuration", "path", configFile)
	} else {
		config.Save()
		logger.Info("Generated configuration file", "path", configFile)
	}

	return nil
}
