package main

import (
	"log/slog"
	"os"
)

func SlogFatal(msg string, args ...any) {
	slog.Error(msg, args...)
	exit(1)
}

func setupLogging(opts Opts) {
	var programLevel = new(slog.LevelVar)
	hOpts := &slog.HandlerOptions{Level: programLevel}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, hOpts)))
	if opts.Verbose {
		programLevel.Set(slog.LevelDebug)
	}
}
