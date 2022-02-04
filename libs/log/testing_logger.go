package log

import (
	"io"
	"os"
	"testing"

	"github.com/go-kit/log/term"
)

var (
	// reuse the same logger across all tests
	_testingLogger Logger
)

// TestingLogger returns a OCLogger which writes to STDOUT if testing being run
// with the verbose (-v) flag, NopLogger otherwise.
//
// Note that the call to TestingLogger() must be made
// inside a test (not in the init func) because
// verbose flag only set at the time of testing.
func TestingLogger() Logger {
	return TestingLoggerWithOutput(os.Stdout)
}

// TestingLoggerWOutput returns a OCLogger which writes to (w io.Writer) if testing being run
// with the verbose (-v) flag, NopLogger otherwise.
//
// Note that the call to TestingLoggerWithOutput(w io.Writer) must be made
// inside a test (not in the init func) because
// verbose flag only set at the time of testing.
func TestingLoggerWithOutput(w io.Writer) Logger {
	if _testingLogger != nil {
		return _testingLogger
	}

	if testing.Verbose() {
		_testingLogger = NewOCLogger(NewSyncWriter(w))
	} else {
		_testingLogger = NewNopLogger()
	}

	return _testingLogger
}

// TestingLoggerWithColorFn allow you to provide your own color function. See
// TestingLogger for documentation.
func TestingLoggerWithColorFn(colorFn func(keyvals ...interface{}) term.FgBgColor) Logger {
	if _testingLogger != nil {
		return _testingLogger
	}

	if testing.Verbose() {
		_testingLogger = NewOCLoggerWithColorFn(NewSyncWriter(os.Stdout), colorFn)
	} else {
		_testingLogger = NewNopLogger()
	}

	return _testingLogger
}
