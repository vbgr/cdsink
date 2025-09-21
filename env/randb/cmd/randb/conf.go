package main

import (
	"log/slog"
	"slices"

	"github.com/asaskevich/govalidator"

	"randb/pkg/randb"
	"randb/pkg/driver"
)

const (
	logFormatText = "TEXT"
	logFormatJson = "JSON"
)

var (
	logLevels = []string{
		"INFO",
		"WARNING",
		"ERROR",
		"DEBUG",
	}

	slogLevels = []slog.Level{
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
		slog.LevelDebug,
	}

	logFormats = []string{
		logFormatText,
		logFormatJson,
	}
)

func newEnumValidator(enum []string) govalidator.CustomTypeValidator {
	return func(i, o interface{}) bool {
		switch v := i.(type) {
		case string:
			return slices.Contains(enum, v)
		default:
			return false
		}
	}
}

func init() {
	govalidator.CustomTypeTagMap.Set("log-fmt", newEnumValidator(logFormats))
	govalidator.CustomTypeTagMap.Set("log-lvl", newEnumValidator(logLevels))
}

type Log struct {
	Level  string `valid:"log-lvl~invalid log level provided"`
	Format string `valid:"log-fmt~invalid log format provided"`
}

type Config struct {
	Log       Log
	Database  driver.Config
	Generator randb.Config
}
