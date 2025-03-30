package main

import (
	"log/slog"
	"os"
)

func run(args ...string) error {
	cmd := command(args[0], args[1:]...)
	slog.Debug("run", "command", args)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runDevNull(args ...string) error {
	cmd := command(args[0], args[1:]...)
	slog.Debug("run", "command", args)
	return cmd.Run()
}

func main() {
	opts, err := GetOpts(os.Args[1:])
	if err != nil {
		return
	}

	setupLogging(opts)
	slog.Debug("cmdline", "opts", opts)

	cfg, err := GetConfig()
	if err != nil {
		SlogFatal("fatal error", "error", err)
	}

	app := App{Config: cfg, Opts: opts}

	if opts.Launch {
		if err := app.launch(); err != nil {
			SlogFatal("failed to launch", "error", err)
		}
	}

	if err := app.shell(); err != nil {
		SlogFatal("failed to create shell", "error", err)
	}
}
