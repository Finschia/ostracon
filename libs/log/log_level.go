package log

import (
	"errors"
	"fmt"
	"strings"
)

const (
	defaultLogLevelKey = "*"
)

// ParseLogLevel parses complex log level - comma-separated
// list of module:level pairs with an optional *:level pair (* means
// all other modules).
//
// Example:
//
//	ParseLogLevel("consensus:debug,mempool:debug,*:error", NewOCLogger(os.Stdout), "info")
func ParseLogLevel(lvl string, logger Logger, defaultLogLevelValue string) (Logger, error) {
	if lvl == "" {
		return nil, errors.New("empty log level")
	}

	l := lvl

	// prefix simple one word levels (e.g. "info") with "*"
	if !strings.Contains(l, ":") {
		l = defaultLogLevelKey + ":" + l
	}

	options := make([]Option, 0)

	isDefaultLogLevelSet := false
	var option Option
	var err error

	list := strings.Split(l, ",")
	for _, item := range list {
		moduleAndLevel := strings.Split(item, ":")

		if len(moduleAndLevel) != 2 {
			return nil, fmt.Errorf("expected list in a form of \"module:level\" pairs, given pair %s, list %s", item, list)
		}

		module := moduleAndLevel[0]
		level := moduleAndLevel[1]

		if module == defaultLogLevelKey {
			option, err = AllowLevel(level)
			if err != nil {
				return nil, fmt.Errorf("failed to parse default log level (pair %s, list %s): %w", item, l, err)
			}
			options = append(options, option)
			isDefaultLogLevelSet = true
		} else {
			switch level {
			case "debug":
				option = AllowDebugWith("module", module)
			case "info":
				option = AllowInfoWith("module", module)
			case "error":
				option = AllowErrorWith("module", module)
			case "none":
				option = AllowNoneWith("module", module)
			default:
				return nil,
					fmt.Errorf("expected either \"info\", \"debug\", \"error\" or \"none\" log level, given %s (pair %s, list %s)",
						level,
						item,
						list)
			}
			options = append(options, option)

		}
	}

	// if "*" is not provided, set default global level
	if !isDefaultLogLevelSet {
		option, err = AllowLevel(defaultLogLevelValue)
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	}

	return NewFilter(logger, options...), nil
}
