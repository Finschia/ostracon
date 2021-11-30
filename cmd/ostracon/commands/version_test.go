package commands

import (
	"github.com/line/ostracon/version"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	version.OCCoreSemVer = "test version"
	VersionCmd.Run(VersionCmd, nil)
}
