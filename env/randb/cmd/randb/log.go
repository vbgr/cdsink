package main

import (
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/lmittmann/tint"
)

func setupLog(l Log) error {
	if _, err := govalidator.ValidateStruct(&l); err != nil {
		return err
	}

	level := slogLevels[slices.Index(logLevels, l.Level)]

	if logFormats[slices.Index(logFormats, l.Format)] == logFormatJson {
		opts := slog.HandlerOptions{Level: level}
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &opts)))
	} else {
		opts := tint.Options{Level: level, TimeFormat: time.RFC3339}
		slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &opts)))
	}

	return nil
}
