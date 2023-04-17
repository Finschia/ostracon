package commands

import (
	"testing"

	"github.com/Finschia/ostracon/version"
)

func TestVersionCmd(t *testing.T) {
	version.OCCoreSemVer = "test version"
	VersionCmd.Run(VersionCmd, nil)
}
