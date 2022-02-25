package log

import (
	"io"

	kitlog "github.com/go-kit/log"
)

// NewOCJSONLogger returns a Logger that encodes keyvals to the Writer as a
// single JSON object. Each log event produces no more than one call to
// w.Write. The passed Writer must be safe for concurrent use by multiple
// goroutines if the returned Logger will be used concurrently.
func NewOCJSONLogger(w io.Writer) Logger {
	logger := kitlog.NewJSONLogger(w)
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
	return &ocLogger{logger}
}

// NewOCJSONLoggerNoTS is the same as NewOCJSONLogger, but without the
// timestamp.
func NewOCJSONLoggerNoTS(w io.Writer) Logger {
	logger := kitlog.NewJSONLogger(w)
	return &ocLogger{logger}
}
