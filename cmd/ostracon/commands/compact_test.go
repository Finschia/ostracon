package commands

import (
	cfg "github.com/Finschia/ostracon/config"
	"github.com/Finschia/ostracon/libs/log"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test_RunE(t *testing.T) {
	config = cfg.TestConfig()
	config.RootDir = os.TempDir()
	config.DBBackend = "badgerdb" // not support
	logger = log.TestingLogger()
	err := CompactGoLevelDBCmd.RunE(nil, nil)
	assert.Error(t, err)
	config.DBBackend = "goleveldb"
	err = CompactGoLevelDBCmd.RunE(nil, nil)
	assert.NoError(t, err)
}

func Test_compactGoLevelDBs(t *testing.T) {
	type args struct {
		rootDir string
		logger  log.Logger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "success",
			args: args{
				rootDir: os.TempDir(),
				logger:  log.TestingLogger(),
			},
		},
		{
			name: "doesn't exist db dir",
			args: args{
				rootDir: "/nonexistent",
				logger:  log.TestingLogger(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compactGoLevelDBs(tt.args.rootDir, tt.args.logger)
		})
	}
}
