package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/line/ostracon/libs/log"
	tmos "github.com/line/ostracon/libs/os"
	"github.com/line/ostracon/privval"
)

// ResetAllCmd removes the database of this Ostracon core
// instance.
var ResetAllCmd = &cobra.Command{
	Use:     "unsafe-reset-all",
	Aliases: []string{"unsafe_reset_all"},
	Short:   "(unsafe) Remove all the data and WAL, reset this node's validator to genesis state",
	Run:     resetAll,
	PreRun:  deprecateSnakeCase,
}

var keepAddrBook bool

func init() {
	ResetAllCmd.Flags().BoolVar(&keepAddrBook, "keep-addr-book", false, "keep the address book intact")
	ResetAllCmd.Flags().String("priv_key_type", config.PrivKeyType,
		"Specify validator's private key type (ed25519 | composite)")
	ResetPrivValidatorCmd.Flags().String("priv_key_type", config.PrivKeyType,
		"Specify validator's private key type (ed25519 | composite)")
}

// ResetPrivValidatorCmd resets the private validator files.
var ResetPrivValidatorCmd = &cobra.Command{
	Use:     "unsafe-reset-priv-validator",
	Aliases: []string{"unsafe_reset_priv_validator"},
	Short:   "(unsafe) Reset this node's validator to genesis state",
	Run:     resetPrivValidator,
	PreRun:  deprecateSnakeCase,
}

// XXX: this is totally unsafe.
// it's only suitable for testnets.
func resetAll(cmd *cobra.Command, args []string) {
	ResetAll(config.DBDir(), config.P2P.AddrBookFile(), config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(), config.PrivValidatorKeyType(), logger)
}

// XXX: this is totally unsafe.
// it's only suitable for testnets.
func resetPrivValidator(cmd *cobra.Command, args []string) {
	resetFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile(),
		config.PrivValidatorKeyType(), logger)
}

// ResetAll removes address book files plus all data, and resets the privValdiator data.
// Exported so other CLI tools can use it.
func ResetAll(dbDir, addrBookFile, privValKeyFile, privValStateFile, privKeyType string, logger log.Logger) error {
	if keepAddrBook {
		logger.Info("The address book remains intact")
	} else {
		removeAddrBook(addrBookFile, logger)
	}
	if err := os.RemoveAll(dbDir); err == nil {
		logger.Info("Removed all blockchain history", "dir", dbDir)
	} else {
		logger.Error("Error removing all blockchain history", "dir", dbDir, "err", err)
	}
	// recreate the dbDir since the privVal state needs to live there
	if err := tmos.EnsureDir(dbDir, 0700); err != nil {
		logger.Error("unable to recreate dbDir", "err", err)
	}
	return resetFilePV(privValKeyFile, privValStateFile, privKeyType, logger)
}

func resetFilePV(privValKeyFile, privValStateFile, privKeyType string, logger log.Logger) error {
	if _, err := os.Stat(privValKeyFile); err == nil {
		pv := privval.LoadFilePVEmptyState(privValKeyFile, privValStateFile)
		pv.Reset()
		logger.Info("Reset private validator file to genesis state", "keyFile", privValKeyFile,
			"stateFile", privValStateFile)
	} else {
		pv, err := privval.GenFilePV(privValKeyFile, privValStateFile, privKeyType)
		if err != nil {
			return err
		}
		pv.Save()
		logger.Info("Generated private validator file", "keyFile", privValKeyFile,
			"stateFile", privValStateFile)
	}
	return nil
}

func removeAddrBook(addrBookFile string, logger log.Logger) {
	if err := os.Remove(addrBookFile); err == nil {
		logger.Info("Removed existing address book", "file", addrBookFile)
	} else if !os.IsNotExist(err) {
		logger.Info("Error removing address book", "file", addrBookFile, "err", err)
	}
}
