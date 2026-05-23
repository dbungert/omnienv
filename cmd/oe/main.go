package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
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

func Run() error {
	opts, err := GetOpts(os.Args[1:])
	if err != nil {
		return nil
	}

	if opts.Version {
		ver := "unknown"
		if bi, ok := debug.ReadBuildInfo(); ok {
			ver = bi.Main.Version
		}
		fmt.Printf("omnienv version: %v\n", ver)
		return nil
	}

	setupLogging(opts)
	slog.Debug("cmdline", "opts", opts)

	cfg, err := GetConfig()
	if err != nil {
		return fmt.Errorf("fatal error: %w", err)
	}

	app := App{Config: cfg, Opts: opts}

	if opts.Launch {
		if err := app.launch(); err != nil {
			return fmt.Errorf("failed to launch: %w", err)
		}
	}

	if err := app.shell(); err != nil {
		return fmt.Errorf("failed to create shell: %w", err)
	}

	return nil
}

func main() {
	if err := Run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}
