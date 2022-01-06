package commands

import (
	"testing"

	"github.com/line/ostracon/version"
)

func TestVersionCmd(t *testing.T) {
	version.OCCoreSemVer = "test version"
	VersionCmd.Run(VersionCmd, nil)
}
