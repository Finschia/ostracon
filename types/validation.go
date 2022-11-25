package types

import (
	"fmt"
	"time"

	"github.com/line/ostracon/crypto/tmhash"
	"github.com/line/ostracon/crypto/vrf"
	tmtime "github.com/line/ostracon/types/time"
)

// ValidateTime does a basic time validation ensuring time does not drift too
// much: +/- one year.
// TODO: reduce this to eg 1 day
// NOTE: DO NOT USE in ValidateBasic methods in this package. This function
// can only be used for real time validation, like on proposals and votes
// in the consensus. If consensus is stuck, and rounds increase for more than a day,
// having only a 1-day band here could break things...
// Can't use for validating blocks because we may be syncing years worth of history.
func ValidateTime(t time.Time) error {
	var (
		now     = tmtime.Now()
		oneYear = 8766 * time.Hour
	)
	if t.Before(now.Add(-oneYear)) || t.After(now.Add(oneYear)) {
		return fmt.Errorf("time drifted too much. Expected: -1 < %v < 1 year", now)
	}
	return nil
}

// ValidateHash returns an error if the hash is not empty, but its
// size != tmhash.Size.
func ValidateHash(h []byte) error {
	if len(h) > 0 && len(h) != tmhash.Size {
		return fmt.Errorf("expected size to be %d bytes, got %d bytes",
			tmhash.Size,
			len(h),
		)
	}
	return nil
}

// ValidateProof returns an error if the proof is not empty, but its
// size != vrf.ProofLength().
func ValidateProof(h []byte) error {
	if len(h) > 0 && len(h) != vrf.ProofLength() {
		return fmt.Errorf("expected size to be %d bytes, got %d bytes",
			vrf.ProofLength(),
			len(h),
		)
	}
	return nil
}
