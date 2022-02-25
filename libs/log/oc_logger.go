package log

import (
	"fmt"
	"io"

	kitlog "github.com/go-kit/log"
	kitlevel "github.com/go-kit/log/level"
	"github.com/go-kit/log/term"
)

const (
	msgKey    = "_msg" // "_" prefixed to avoid collisions
	moduleKey = "module"
)

type ocLogger struct {
	srcLogger kitlog.Logger
}

// Interface assertions
var _ Logger = (*ocLogger)(nil)

// NewOCLogger returns a logger that encodes msg and keyvals to the Writer
// using go-kit's log as an underlying logger and our custom formatter. Note
// that underlying logger could be swapped with something else.
func NewOCLogger(w io.Writer) Logger {
	// Color by level value
	colorFn := func(keyvals ...interface{}) term.FgBgColor {
		if keyvals[0] != kitlevel.Key() {
			panic(fmt.Sprintf("expected level key to be first, got %v", keyvals[0]))
		}
		switch keyvals[1].(kitlevel.Value).String() {
		case "debug":
			return term.FgBgColor{Fg: term.DarkGray}
		case "error":
			return term.FgBgColor{Fg: term.Red}
		default:
			return term.FgBgColor{}
		}
	}

	return &ocLogger{term.NewLogger(w, NewOCFmtLogger, colorFn)}
}

// NewOCLoggerWithColorFn allows you to provide your own color function. See
// NewOCLogger for documentation.
func NewOCLoggerWithColorFn(w io.Writer, colorFn func(keyvals ...interface{}) term.FgBgColor) Logger {
	return &ocLogger{term.NewLogger(w, NewOCFmtLogger, colorFn)}
}

// Info logs a message at level Info.
func (l *ocLogger) Info(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Info(l.srcLogger)

	if err := kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...); err != nil {
		errLogger := kitlevel.Error(l.srcLogger)
		kitlog.With(errLogger, msgKey, msg).Log("err", err) //nolint:errcheck // no need to check error again
	}
}

// Debug logs a message at level Debug.
func (l *ocLogger) Debug(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Debug(l.srcLogger)

	if err := kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...); err != nil {
		errLogger := kitlevel.Error(l.srcLogger)
		kitlog.With(errLogger, msgKey, msg).Log("err", err) //nolint:errcheck // no need to check error again
	}
}

// Error logs a message at level Error.
func (l *ocLogger) Error(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Error(l.srcLogger)

	lWithMsg := kitlog.With(lWithLevel, msgKey, msg)
	if err := lWithMsg.Log(keyvals...); err != nil {
		lWithMsg.Log("err", err) //nolint:errcheck // no need to check error again
	}
}

// With returns a new contextual logger with keyvals prepended to those passed
// to calls to Info, Debug or Error.
func (l *ocLogger) With(keyvals ...interface{}) Logger {
	return &ocLogger{kitlog.With(l.srcLogger, keyvals...)}
}
