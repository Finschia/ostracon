package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/privval"
)

// GenValidatorCmd allows the generation of a keypair for a
// validator.
var GenValidatorCmd = &cobra.Command{
	Use:     "gen-validator",
	Aliases: []string{"gen_validator"},
	Short:   "Generate new validator keypair",
	PreRun:  deprecateSnakeCase,
	Run:     genValidator,
}

func init() {
	GenValidatorCmd.Flags().String("priv_key_type", config.PrivKeyType,
		"Specify validator's private key type (ed25519 | composite)")
}

func genValidator(cmd *cobra.Command, args []string) {
	pv, _ := privval.GenFilePV("", "", config.PrivKeyType)
	jsbz, err := tmjson.Marshal(pv)
	if err != nil {
		panic(err)
	}
	fmt.Printf(`%v
`, string(jsbz))
}
