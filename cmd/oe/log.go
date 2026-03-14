package main

import (
	"log/slog"
)

func setupLogging(opts Opts) {
	var programLevel = new(slog.LevelVar)
	hOpts := &slog.HandlerOptions{Level: programLevel}
	slog.SetDefault(slog.New(slog.NewTextHandler(stderr, hOpts)))
	if opts.Verbose {
		programLevel.Set(slog.LevelDebug)
	}
}
