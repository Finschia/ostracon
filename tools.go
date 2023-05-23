//go:build tools
// +build tools

package tools

// NOTE: This import lists packages for tools that aren't used in production code but are needed for development, such
// as static checking or code generation. This is to prevent such packages from being erased from `go.mod` by `go mod
// tidy` or dependabot.
import (
	_ "github.com/vektra/mockery/v2"
)
