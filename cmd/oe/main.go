package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"al.essio.dev/pkg/shellescape"
	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

func run(args ...string) error {
	cmd := command(args[0], args[1:]...)
	slog.Debug("run", "command", args)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type App struct {
	Config
	Opts
}

func (app App) start(c lxd.InstanceServer) error {
	reqState := api.InstanceStatePut{Action: "start", Timeout: -1}
	op, err := c.UpdateInstanceState(app.Config.Name(), reqState, "")
	if err != nil {
		return err
	}

	if err = op.Wait(); err != nil {
		return err
	}

	return nil
}

func (app App) startIfNeeded() error {
	// Connect to LXD over the Unix socket
	c, err := connectLXDUnix("", nil)
	if err != nil {
		return err
	}

	// middle arg is the etag
	state, _, err := c.GetInstanceState(app.Config.Name())
	if err != nil {
		return err
	}

	slog.Debug("startIfNeeded", "instanceStatus", state.Status)
	switch state.Status {
	case "Stopped":
		return app.start(c)
	case "Running":
		// no action required
		return nil
	default:
		return fmt.Errorf("no handler for Status %v", state.Status)
	}
}

func (app App) wait() error {
	if !app.Config.IsVM() {
		return nil
	}
	for {
		err := run("lxc", "exec", app.Config.Name(), "--", "/bin/true")
		if err == nil {
			return nil
		}

		exitError, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}

		if ec := exitError.ExitCode(); ec != 255 {
			return fmt.Errorf("strange exit code %d", ec)
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (app App) launch() error {
	image := "ubuntu-daily:" + app.Config.Series
	args := []string{"lxc", "launch", image, app.Config.Name()}
	if app.Config.IsVM() {
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
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := app.wait(); err != nil {
		return fmt.Errorf("failed to wait for instance: %w", err)
	}

	cloud_init := []string{
		"lxc", "exec", app.Config.Name(), "--",
		"cloud-init", "status", "--wait",
	}
	if err := run(cloud_init...); err != nil {
		return fmt.Errorf("cloud-init failure: %w", err)
	}
	return nil
}

func (app App) lxcExec(script string) error {
	lxc, err := lookPath("lxc")
	if err != nil {
		return err
	}

	args := []string{
		// get a shell to the instance via lxc
		lxc, "exec", app.Config.Name(), "--",

		// login as $USER, get a pty, run script
		"su", "-P", "-", os.Getenv("USER"), "-c", script,

		// su -P only works sometimes
		//   22.04: jammy+ is fine
		//   20.04: focal fails the user ownership of /dev/pts/2
		//   18.04: bionic su has no "-P", at least not build-time
		//          enabled, and might have the same focal problems if
		//          we rebuilt
		//
		// should I run something like
		//   https://github.com/creack/pty on the other side?
	}
	slog.Debug("exec", "command", args)
	envv := os.Environ()
	return syscallExec(args[0], args, envv)
}

func (app App) shell() error {
	if err := app.startIfNeeded(); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	if err := app.wait(); err != nil {
		return fmt.Errorf("failed to wait for instance: %w", err)
	}

	// in instance, change to the directory we are in right now
	script := fmt.Sprintf(`cd "%s" && exec $SHELL`, os.Getenv("PWD"))
	if len(app.Opts.Params) > 1 {
		// run shell with the command we were given
		script = fmt.Sprintf(
			`%s -c "%s"`, script,
			shellescape.QuoteCommand(app.Opts.Params[1:]),
		)
	}

	if err := app.lxcExec(script); err != nil {
		return fmt.Errorf("failed to lxc exec: %w", err)
	}
	return nil
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
