package main

import (
	"log/slog"
)

func setupLogging(verbose bool) {
	var programLevel = new(slog.LevelVar)
	hOpts := &slog.HandlerOptions{Level: programLevel}
	slog.SetDefault(slog.New(slog.NewTextHandler(stderr, hOpts)))
	if verbose {
		programLevel.Set(slog.LevelDebug)
	}
}
