package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"

	"al.essio.dev/pkg/shellescape"
	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
)

type App struct {
	Config Config
	Opts   Opts
}

func (app App) suCanPty() bool {
	floatVal, err := strconv.ParseFloat(app.system(), 64)
	if err == nil {
		// jammy (22.04 is fine)
		return floatVal > 22.039
	}

	switch app.system() {
	case "jammy", "noble", "plucky":
		return true
	default:
		return false
	}
}

func (app App) launchImage() string {
	if app.Opts.System != "" {
		return NewSystem(app.Opts.System).LaunchImage()
	}
	return app.Config.System.LaunchImage()
}

func (app App) system() string {
	if app.Opts.System != "" {
		return app.Opts.System
	}
	return app.Config.System.Name
}

func (app App) name() string {
	return fmt.Sprintf("%s-%s", app.Config.Label, app.system())
}

func (app App) start(c lxd.InstanceServer) error {
	reqState := api.InstanceStatePut{Action: "start", Timeout: -1}
	op, err := c.UpdateInstanceState(app.name(), reqState, "")
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
	defer c.Disconnect()

	// middle arg is the etag
	state, _, err := c.GetInstanceState(app.name())
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

func (app App) isVM() (bool, error) {
	// Connect to LXD over the Unix socket
	c, err := connectLXDUnix("", nil)
	if err != nil {
		return false, err
	}
	defer c.Disconnect()

	// middle arg is the etag
	inst, _, err := c.GetInstance(app.name())
	if err != nil {
		return false, err
	}

	return inst.Type == "virtual-machine", nil
}

func (app App) wait() error {
	vm, err := app.isVM()
	if err != nil {
		return err
	}
	if !vm {
		return nil
	}

	fmt.Print("Waiting.")
	for {
		err := runDevNull(
			"lxc", "exec", app.name(), "--", "/bin/true",
		)
		if err == nil {
			fmt.Println()
			return nil
		}

		exitError, ok := err.(*exec.ExitError)
		if !ok {
			return err
		}

		if ec := exitError.ExitCode(); ec != 255 {
			return fmt.Errorf("strange exit code %d", ec)
		} else {
			fmt.Print(".")
			timeSleep(time.Second)
		}
	}
}

func (app App) lxcRun(args ...string) error {
	cmd := append([]string{"lxc", "exec", app.name(), "--"}, args...)
	if err := run(cmd...); err != nil {
		return fmt.Errorf("cmd %v failure: %w", cmd, err)
	}
	return nil
}

func (app App) lp1878225Quirk() error {
	// LP: #1878225 - cloud-init status --wait appears to never resolve, as
	// other things earlier in the chain aren't finalized.  Per the LP,
	// there are problems having snapd seeded complete, and this is
	// apparently severe enough to trigger no longer seeding lxd as a snap
	// in subsequent releases.  Only Jammy appears affected among the
	// tested images.

	script := `
	command -v lsb_release || exit 0
	[ "$(lsb_release -i -s)" = "Ubuntu" ] || exit 0
	[ "$(lsb_release -r -s)" = "22.04" ] || exit 0
	exit 225
	`
	args := []string{"lxc", "exec", app.name(), "--", "sh", "-c", script}
	slog.Debug("run", "command", args)
	cmd := command(args[0], args[1:]...)
	err := cmd.Run()
	if err == nil {
		slog.Debug("skipping LP: #1878225 quirk")
		return nil
	}

	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return err
	}

	if ec := exitError.ExitCode(); ec != 225 {
		return fmt.Errorf("strange exit code %d", ec)
	}

	// often systemctl fails here due to the socket not being up yet,
	// so wait for that first
	script = `
	for i in $(seq 10); do
	    if [ -e /run/dbus/system_bus_socket ]; then
	        exit 0
	    fi
	    sleep 1
	done
	[ -e /run/dbus/system_bus_socket ]
	`

	if err := app.lxcRun("sh", "-c", script); err != nil {
		return fmt.Errorf("bus wait failure: %w", err)
	}

	// the actual workaround
	if err := app.lxcRun("systemctl", "stop", "snapd.seeded.service"); err != nil {
		return fmt.Errorf("seeded stop failure: %w", err)
	}

	return nil
}

func (app App) launch() error {
	args := []string{"lxc", "launch", app.launchImage(), app.name()}
	if app.Config.IsVM() {
		args = append(args, "--vm")
	}

	cmd := command(args[0], args[1:]...)
	slog.Debug("run", "command", args)
	cmd.Stdout = os.Stdout
	cmd.Stdin = bytes.NewReader([]byte(app.Config.LXDLaunchConfig()))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := app.wait(); err != nil {
		return fmt.Errorf("failed to wait for instance: %w", err)
	}

	use_pty := []string{
		"sh", "-c", "echo 'Defaults use_pty' > /etc/sudoers.d/use_pty",
	}
	if err := app.lxcRun(use_pty...); err != nil {
		return fmt.Errorf("use_pty setup failure: %w", err)
	}

	if err := app.lp1878225Quirk(); err != nil {
		return fmt.Errorf("LP #1878225 workaround failure: %w", err)
	}

	if err := app.lxcRun("cloud-init", "status", "--wait"); err != nil {
		return fmt.Errorf("cloud-init failure: %w", err)
	}
	return nil
}

func (app App) sudoLogin(script string) []string {
	return []string{
		"sudo", "--login", "--user", os.Getenv("USER"),
		"sh", "-c", script,
	}
}

func (app App) lxcExec(script string) error {
	lxc, err := lookPath("lxc")
	if err != nil {
		return err
	}

	// get a shell to the instance via lxc
	args := []string{lxc, "exec", app.name(), "--"}
	args = append(args, app.sudoLogin(script)...)

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
	if len(app.Opts.Params) > 0 {
		// run shell with the command we were given
		script = fmt.Sprintf(
			`%s -c "%s"`, script,
			shellescape.QuoteCommand(app.Opts.Params),
		)
	}

	if err := app.lxcExec(script); err != nil {
		return fmt.Errorf("failed to lxc exec: %w", err)
	}
	return nil
}
