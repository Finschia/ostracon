package types

import "fmt"

type (
	// ErrInvalidCommitHeight is returned when we encounter a commit with an
	// unexpected height.
	ErrInvalidCommitHeight struct {
		Expected int64
		Actual   int64
	}

	// ErrInvalidCommitSignatures is returned when we encounter a commit where
	// the number of signatures doesn't match the number of validators.
	ErrInvalidCommitSignatures struct {
		Expected int
		Actual   int
	}

	// ErrUnsupportedKey is returned when we encounter a private key which doesn't
	// support generating VRF proof.
	ErrUnsupportedKey struct {
		Expected string
	}

	// VRF verification failure
	ErrInvalidProof struct {
		ErrorMessage string
	}

	// invalid round
	ErrInvalidRound struct {
		ConsensusRound int32
		BlockRound     int32
	}
)

func NewErrInvalidCommitHeight(expected, actual int64) ErrInvalidCommitHeight {
	return ErrInvalidCommitHeight{
		Expected: expected,
		Actual:   actual,
	}
}

func (e ErrInvalidCommitHeight) Error() string {
	return fmt.Sprintf("Invalid commit -- wrong height: %v vs %v", e.Expected, e.Actual)
}

func NewErrInvalidCommitSignatures(expected, actual int) ErrInvalidCommitSignatures {
	return ErrInvalidCommitSignatures{
		Expected: expected,
		Actual:   actual,
	}
}

func (e ErrInvalidCommitSignatures) Error() string {
	return fmt.Sprintf("Invalid commit -- wrong set size: %v vs %v", e.Expected, e.Actual)
}

func NewErrUnsupportedKey(expected string) ErrUnsupportedKey {
	return ErrUnsupportedKey{
		Expected: expected,
	}
}

func (e ErrUnsupportedKey) Error() string {
	return fmt.Sprintf("the private key is not a %s", e.Expected)
}

func NewErrInvalidProof(message string) ErrInvalidProof {
	return ErrInvalidProof{ErrorMessage: message}
}

func (e ErrInvalidProof) Error() string {
	return fmt.Sprintf("Proof verification failed: %s", e.ErrorMessage)
}

func NewErrInvalidRound(consensusRound, blockRound int32) ErrInvalidRound {
	return ErrInvalidRound{ConsensusRound: consensusRound, BlockRound: blockRound}
}

func (e ErrInvalidRound) Error() string {
	return fmt.Sprintf("Block round(%d) is mismatched to consensus round(%d)", e.BlockRound, e.ConsensusRound)
}
