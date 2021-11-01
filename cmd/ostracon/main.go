package main

import (
	"os"
	"path/filepath"

	cmd "github.com/line/ostracon/cmd/ostracon/commands"
	"github.com/line/ostracon/cmd/ostracon/commands/debug"
	cfg "github.com/line/ostracon/config"
	"github.com/line/ostracon/libs/cli"
	nm "github.com/line/ostracon/node"
)

func main() {
	rootCmd := cmd.RootCmd
	rootCmd.AddCommand(
		cmd.GenValidatorCmd,
		cmd.ProbeUpnpCmd,
		cmd.LightCmd,
		cmd.ReplayCmd,
		cmd.ReplayConsoleCmd,
		cmd.ResetAllCmd,
		cmd.ResetPrivValidatorCmd,
		cmd.ShowValidatorCmd,
		cmd.TestnetFilesCmd,
		cmd.ShowNodeIDCmd,
		cmd.GenNodeKeyCmd,
		cmd.VersionCmd,
		debug.DebugCmd,
		cli.NewCompletionCmd(rootCmd, true),
	)

	// NOTE:
	// Users wishing to:
	//	* Use an external signer for their validators
	//	* Supply an in-proc abci app
	//	* Supply a genesis doc file from another source
	//	* Provide their own DB implementation
	// can copy this file and use something other than the
	// DefaultNewNode function
	nodeFunc := nm.DefaultNewNode

	// Create & start node
	rootCmd.AddCommand(cmd.NewInitCmd())
	rootCmd.AddCommand(cmd.NewRunNodeCmd(nodeFunc))

	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	cmd := cli.PrepareBaseCmd(rootCmd, "OC", filepath.Join(userHome, cfg.DefaultOstraconDir))
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
