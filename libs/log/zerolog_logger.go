package log

import (
	"fmt"
	"io"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var _ Logger = (*ZeroLogWrapper)(nil)

// ZeroLogWrapper provides a wrapper around a zerolog.Logger instance. It implements
// Tendermint's Logger interface.
type ZeroLogWrapper struct {
	zerolog.Logger
}

type ZeroLogConfig struct {
	IsLogPlain bool
	LogLevel   string

	LogPath       string
	LogMaxAge     int
	LogMaxSize    int
	LogMaxBackups int
}

func NewZeroLogConfig(isLogPlain bool, logLevel string, logPath string, logMaxAge int, logMaxSize int, logMaxBackups int) ZeroLogConfig {
	return ZeroLogConfig{
		IsLogPlain:    isLogPlain,
		LogLevel:      logLevel,
		LogPath:       logPath,
		LogMaxAge:     logMaxAge,
		LogMaxSize:    logMaxSize,
		LogMaxBackups: logMaxBackups,
	}
}

func NewZeroLogLogger(cfg ZeroLogConfig, consoleWriter io.Writer) (Logger, error) {
	var logWriter io.Writer
	if cfg.IsLogPlain {
		logWriter = zerolog.ConsoleWriter{
			Out:        consoleWriter,
			TimeFormat: "2006/01/02-15:04:05.999",
		}
	} else {
		logWriter = consoleWriter
	}

	var zeroLogLogger zerolog.Logger
	if cfg.LogPath != "" {
		// initialize the rotator
		rotator := &lumberjack.Logger{
			Filename:   cfg.LogPath,
			MaxAge:     cfg.LogMaxAge,
			MaxSize:    cfg.LogMaxSize,
			MaxBackups: cfg.LogMaxBackups,
		}

		// set log format
		var fileLogWriter io.Writer
		if cfg.IsLogPlain {
			fileLogWriter = zerolog.ConsoleWriter{
				Out:        rotator,
				NoColor:    true,
				TimeFormat: "2006/01/02-15:04:05.999",
			}
		} else {
			fileLogWriter = rotator
		}

		zeroLogLogger = zerolog.New(zerolog.MultiLevelWriter(logWriter, fileLogWriter)).With().Timestamp().Logger()
	} else {
		zeroLogLogger = zerolog.New(logWriter).With().Timestamp().Logger()
	}

	leveledZeroLogLogger, err := ParseLogLevel(cfg.LogLevel, ZeroLogWrapper{zeroLogLogger}, "info")
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level (%s): %w", cfg.LogLevel, err)
	}

	return leveledZeroLogLogger, nil
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
}

// Info implements Tendermint's Logger interface and logs with level INFO. A set
// of key/value tuples may be provided to add context to the log. The number of
// tuples must be even and the key of the tuple must be a string.
func (z ZeroLogWrapper) Info(msg string, keyVals ...interface{}) {
	z.Logger.Info().Fields(getLogFields(keyVals...)).Msg(msg)
}

// Error implements Tendermint's Logger interface and logs with level ERR. A set
// of key/value tuples may be provided to add context to the log. The number of
// tuples must be even and the key of the tuple must be a string.
func (z ZeroLogWrapper) Error(msg string, keyVals ...interface{}) {
	z.Logger.Error().Fields(getLogFields(keyVals...)).Msg(msg)
}

// Debug implements Tendermint's Logger interface and logs with level DEBUG. A set
// of key/value tuples may be provided to add context to the log. The number of
// tuples must be even and the key of the tuple must be a string.
func (z ZeroLogWrapper) Debug(msg string, keyVals ...interface{}) {
	z.Logger.Debug().Fields(getLogFields(keyVals...)).Msg(msg)
}

// With returns a new wrapped logger with additional context provided by a set
// of key/value tuples. The number of tuples must be even and the key of the
// tuple must be a string.
func (z ZeroLogWrapper) With(keyVals ...interface{}) Logger {
	return ZeroLogWrapper{z.Logger.With().Fields(getLogFields(keyVals...)).Logger()}
}

func getLogFields(keyVals ...interface{}) map[string]interface{} {
	if len(keyVals)%2 != 0 {
		return nil
	}

	fields := make(map[string]interface{})
	for i := 0; i < len(keyVals); i += 2 {
		fields[keyVals[i].(string)] = keyVals[i+1]
	}

	return fields
}
