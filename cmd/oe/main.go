package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"al.essio.dev/pkg/shellescape"
	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

func start(c lxd.InstanceServer, cfg Config) error {
	reqState := api.InstanceStatePut{Action: "start", Timeout: -1}
	op, err := c.UpdateInstanceState(cfg.Name(), reqState, "")
	if err != nil {
		return err
	}

	if err = op.Wait(); err != nil {
		return err
	}

	return nil
}

func startIfNeeded(cfg Config) error {
	// Connect to LXD over the Unix socket
	c, err := connectLXDUnix("", nil)
	if err != nil {
		return err
	}

	// middle arg is the etag
	state, _, err := c.GetInstanceState(cfg.Name())
	if err != nil {
		return err
	}

	slog.Debug("startIfNeeded", "instanceStatus", state.Status)
	switch state.Status {
	case "Stopped":
		return start(c, cfg)
	case "Running":
		// no action required
		return nil
	default:
		return fmt.Errorf("no handler for Status %v", state.Status)
	}
}

func run(args ...string) error {
	cmd := command(args[0], args[1:]...)
	slog.Debug("run", "command", args)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func check(args ...string) {
	if err := run(args...); err != nil {
		SlogFatal("fatal error", "error", err)
	}
}

func wait(cfg Config) error {
	if !cfg.IsVM() {
		return nil
	}
	for {
		err := run("lxc", "exec", cfg.Name(), "--", "/bin/true")
		if err == nil {
			slog.Debug("run check true: no err")
			return nil
		}

		exitError, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}

		ec := exitError.ExitCode()
		slog.Debug("run check true", "ec", ec)
		if ec != 255 {
			return fmt.Errorf("strange exit code %d", ec)
		} else {
			time.Sleep(time.Second)
		}
	}
}

func launch(cfg Config) {
	image := "ubuntu-daily:" + cfg.Series
	args := []string{"lxc", "launch", image, cfg.Name()}
	if cfg.IsVM() {
		args = append(args, "--vm")
	}

	cmd := command(args[0], args[1:]...)
	slog.Debug("run", "command", args)
	cmd.Stdout = os.Stdout
	cmd.Stdin = bytes.NewReader([]byte(`
devices:
  home:
    path: /home
    shift: "true"
    source: /home
    type: disk`))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		SlogFatal("fatal error", "error", err)
	}

	if err := wait(cfg); err != nil {
		SlogFatal("failed to wait for instance", "error", err)
	}

	check("lxc", "exec", cfg.Name(), "--", "cloud-init", "status", "--wait")
}

func shell(cfg Config, opts Opts) {
	if err := startIfNeeded(cfg); err != nil {
		SlogFatal("failed to start instance", "error", err)
	}

	if err := wait(cfg); err != nil {
		SlogFatal("failed to wait for instance", "error", err)
	}

	lxc, err := exec.LookPath("lxc")
	if err != nil {
		SlogFatal("cannot find lxc")
	}

	// in instance, change to the directory we are in right now
	script := fmt.Sprintf(`cd "%s" && exec $SHELL`, os.Getenv("PWD"))

	if len(opts.Params) > 1 {
		// run shell with the command we were given
		script = fmt.Sprintf(
			`%s -c "%s"`, script,
			shellescape.QuoteCommand(opts.Params[1:]),
		)
	}

	args := []string{
		// get a shell to the instance via lxc
		lxc, "exec", cfg.Name(),

		// login as $USER, get a pty, run above script
		"--", "su", "-P", "-", os.Getenv("USER"), "-c", script,

		// su -P only works sometimes
		//   22.04: jammy+ is fine
		//   20.04: focal fails the user ownership of /dev/pts/2
		//   18.04: bionic su has no "-P", at least not build-time
		//          enabled, and might have the same focal problems if
		//          we rebuilt
		//
		// python3 -c 'import pty;pty.spawn("/bin/bash")'
		//   is enough to make bionic pass current tests, but is
		//   fixed to 80 columns
		//
		// should I run something like
		//   https://github.com/creack/pty on the other side?

		// ------------ these are bad plans, don't do these -----------

		// lxc, "shell", cfg.Name(),
		//   this runs things as root in the container, we want to run
		//   as user.  Also, this is a built-in alias for
		//   "exec @ARGS@ -- su -l",

		// lxc, "shell", "--user", "1000", cfg.Name(),
		//   prompts for user credentials

		// lxc, "exec", cfg.Name(),
		// "--", "su", "-", os.Getenv("USER"),
		//   This leaves us with no pty, which means that things that
		//   need it (including the version of subiquity tests that
		//   pass in github) don't work
		//   Manually adding a pty to this does work.

		// lxc, "exec", cfg.Name(),
		// "--", "su", "-p", "-", os.Getenv("USER"),
		//   not even valid

		// lxc, "exec", cfg.Name(),
		// "-t", "--user", "1000", "--", "bash",
		//   busted environment

		// lxc, "exec", cfg.Name(),
		// "-t", "--user", "1000", "--", "bash", "-l",
		//   still a busted environment

		// lxc, "exec", cfg.Name(),
		// "--env", "HOME=/home/dbungert",
		// "-t", "--user", "1000", "--", "bash", "-l",
		//   kind of works?  manually setting HOME is ugly.  something
		//   still broken in environment based on how PS1 prompt is
		//   behaving.  Surely still needs pty.

		// lxc, "exec", cfg.Name(),
		// "-t", "--", "su", "-", os.Getenv("USER"),
		//   I'm not sure what -t does but not passing the pty test

		// lxc, "exec", cfg.Name(),
		// "--cwd", "foo",
		//   might help some cases but su -l ignores this
	}
	slog.Debug("exec", "command", args)
	envv := os.Environ()
	if err := syscall.Exec(args[0], args, envv); err != nil {
		SlogFatal("unexpected return from exec", "err", err)
	}
}

func main() {
	opts, err := GetOpts(os.Args)
	if err != nil {
		return
	}

	setupLogging(opts)
	slog.Debug("cmdline", "opts", opts)

	cfg, err := GetConfig()
	if err != nil {
		SlogFatal("fatal error", "error", err)
	}

	if opts.Launch {
		launch(cfg)
	}

	shell(cfg, opts)
}
