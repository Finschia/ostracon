package evidence

import (
	"github.com/line/ostracon/types"
)

//go:generate mockery --case underscore --name BlockStore

type BlockStore interface {
	LoadBlockMeta(height int64) (*types.BlockMeta, error)
	LoadBlockCommit(height int64) (*types.Commit, error)
	Height() int64
}
