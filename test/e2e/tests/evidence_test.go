package e2e_test

import (
	"bytes"
	"testing"

	e2e "github.com/line/ostracon/test/e2e/pkg"
	"github.com/line/ostracon/types"
)

// assert that all nodes that have blocks at the height of a misbehavior has evidence
// for that misbehavior
func TestEvidence_Misbehavior(t *testing.T) {
	blocks := fetchBlockChain(t)
	testNode(t, func(t *testing.T, node e2e.Node) {
		for _, block := range blocks {
			// Find any evidence blaming this node in this block
			var nodeEvidence types.Evidence
			for _, evidence := range block.Evidence.Evidence {
				switch evidence := evidence.(type) {
				case *types.DuplicateVoteEvidence:
					if bytes.Equal(evidence.VoteA.ValidatorAddress, node.PrivvalKey.PubKey().Address()) {
						nodeEvidence = evidence
					}
				default:
					t.Fatalf("unexpected evidence type %T", evidence)
				}
			}
			if nodeEvidence == nil {
				continue // no evidence for the node at this height
			}
		}
	})
}
