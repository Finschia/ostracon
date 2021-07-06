package mock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/line/ostracon/light/provider"
	"github.com/line/ostracon/types"
)

type Mock struct {
	chainID          string
	headers          map[int64]*types.SignedHeader
	vals             map[int64]*types.ValidatorSet
	voters           map[int64]*types.VoterSet
	evidenceToReport map[string]types.Evidence // hash => evidence
}

var _ provider.Provider = (*Mock)(nil)

// New creates a mock provider with the given set of headers and validator
// sets.
func New(chainID string, headers map[int64]*types.SignedHeader, vals map[int64]*types.ValidatorSet, voters map[int64]*types.VoterSet) *Mock {
	return &Mock{
		chainID:          chainID,
		headers:          headers,
		vals:             vals,
		voters:           voters,
		evidenceToReport: make(map[string]types.Evidence),
	}
}

// ChainID returns the blockchain ID.
func (p *Mock) ChainID() string {
	return p.chainID
}

func (p *Mock) String() string {
	var headers strings.Builder
	for _, h := range p.headers {
		fmt.Fprintf(&headers, " %d:%X", h.Height, h.Hash())
	}

	var voters strings.Builder
	for _, v := range p.vals {
		fmt.Fprintf(&voters, " %X", v.Hash())
	}

	return fmt.Sprintf("Mock{headers: %s, voters: %v}", headers.String(), voters.String())
}

func (p *Mock) LightBlock(_ context.Context, height int64) (*types.LightBlock, error) {
	var lb *types.LightBlock
	if height == 0 && len(p.headers) > 0 {
		sh := p.headers[int64(len(p.headers))]
		vals := p.vals[int64(len(p.vals))]
		voters := p.voters[int64(len(p.voters))]
		lb = &types.LightBlock{
			SignedHeader: sh,
			ValidatorSet: vals,
			VoterSet:     voters,
		}

	}
	if _, ok := p.headers[height]; ok {
		sh := p.headers[height]
		vals := p.vals[height]
		voters := p.voters[height]
		lb = &types.LightBlock{
			SignedHeader: sh,
			ValidatorSet: vals,
			VoterSet:     voters,
		}
	}
	if lb == nil {
		return nil, provider.ErrLightBlockNotFound
	}
	if lb.SignedHeader == nil || lb.ValidatorSet == nil || lb.VoterSet == nil {
		return nil, provider.ErrBadLightBlock{Reason: errors.New("nil header or validators/voters")}
	}
	if err := lb.ValidateBasic(lb.ChainID); err != nil {
		return nil, provider.ErrBadLightBlock{Reason: err}
	}
	return lb, nil
}

func (p *Mock) ReportEvidence(_ context.Context, ev types.Evidence) error {
	p.evidenceToReport[string(ev.Hash())] = ev
	return nil
}

func (p *Mock) HasEvidence(ev types.Evidence) bool {
	_, ok := p.evidenceToReport[string(ev.Hash())]
	return ok
}
