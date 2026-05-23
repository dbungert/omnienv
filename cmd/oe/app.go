package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"al.essio.dev/pkg/shellescape"
)

type App struct {
	Config Config
	Opts   Opts
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

func (app App) start() error {
	args := []string{"lxc", "start", app.name()}
	slog.Debug("run", "command", args)
	if err := run(args...); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	return nil
}

func (app App) startIfNeeded() error {
	cmd := command("lxc", "info", app.name())
	slog.Debug("run", "command", cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get instance info: %w", err)
	}

	var status string
	for _, line := range strings.Split(string(out), "\n") {
		if after, found := strings.CutPrefix(line, "Status: "); found {
			status = after
			break
		}
	}
	if status == "" {
		return fmt.Errorf("could not determine status of instance %s", app.name())
	}

	slog.Debug("startIfNeeded", "instanceStatus", status)
	switch status {
	case "STOPPED":
		return app.start()
	case "RUNNING":
		return nil
	default:
		return fmt.Errorf("no handler for Status %v", status)
	}
}

func (app App) isVM() (bool, error) {
	cmd := command("lxc", "info", app.name())
	slog.Debug("run", "command", cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get instance info: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if after, found := strings.CutPrefix(line, "Type: "); found {
			return after == "virtual-machine", nil
		}
	}
	return false, fmt.Errorf("could not determine type of instance %s", app.name())
}

func (app App) wait() error {
	vm, err := app.isVM()
	if err != nil {
		return err
	}
	if !vm {
		return nil
	}

	fmt.Print("Waiting")
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
			timeSleep(time.Second)
			fmt.Print(".")
		}
	}
}

func (app App) lxcExec(args ...string) error {
	cmd := append([]string{"lxc", "exec", app.name(), "--"}, args...)
	return run(cmd...)
}

func (app App) lxcOutput(args ...string) (string, error) {
	cmd := append([]string{"lxc", "exec", app.name(), "--"}, args...)
	slog.Debug("run", "command", cmd)
	out, err := command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (app App) isUbuntuJammy() (bool, error) {
	// LP: #1878225 - cloud-init status --wait appears to never resolve, as
	// other things earlier in the chain aren't finalized.  Per the LP,
	// there are problems having snapd seeded complete, and this is
	// apparently severe enough to trigger no longer seeding lxd as a snap
	// in subsequent releases.  Only Jammy appears affected among the
	// tested images.

	out, err := app.lxcOutput("lsb_release", "-i", "-s")
	if err != nil || out != "Ubuntu" {
		return false, nil
	}

	out, err = app.lxcOutput("lsb_release", "-r", "-s")
	if err != nil || out != "22.04" {
		return false, nil
	}

	return true, nil
}

func (app App) lp1878225Quirk() error {
	affected, err := app.isUbuntuJammy()
	if err != nil {
		return err
	}
	if !affected {
		slog.Debug("skipping LP: #1878225 quirk")
		return nil
	}

	// often systemctl fails here due to the socket not being up yet,
	// so wait for that first
	script := `
	for i in $(seq 10); do
	    if [ -e /run/dbus/system_bus_socket ]; then
	        exit 0
	    fi
	    sleep 1
	done
	[ -e /run/dbus/system_bus_socket ]
	`

	if err := app.lxcExec("sh", "-c", script); err != nil {
		return fmt.Errorf("bus wait failure: %w", err)
	}

	if err := app.lxcExec("systemctl", "stop", "snapd.seeded.service"); err != nil {
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
	user := CurrentUserInfo()
	cmd.Stdin = bytes.NewReader([]byte(app.Config.LXDLaunchConfig(user)))
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
	if err := app.lxcExec(use_pty...); err != nil {
		return fmt.Errorf("use_pty setup failure: %w", err)
	}

	if err := app.lp1878225Quirk(); err != nil {
		return fmt.Errorf("LP #1878225 workaround failure: %w", err)
	}

	if err := app.lxcExec("cloud-init", "status", "--wait"); err != nil {
		return fmt.Errorf("cloud-init failure: %w", err)
	}
	return nil
}

func (app App) sudoLogin(script string) []string {
	return []string{
		"sudo", "--login", "--user", "user",
		"sh", "-c", script,
	}
}

func (app App) shell() error {
	if err := app.startIfNeeded(); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	if err := app.wait(); err != nil {
		return fmt.Errorf("failed to wait for instance: %w", err)
	}

	// determine where we are relative to RootDir, then adjust that
	// subdirectory against /project, and cd to that
	dest := "/project"
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	if after, found := strings.CutPrefix(wd, app.Config.RootDir); found {
		dest = fmt.Sprintf("%s%s", dest, after)
	}

	script := fmt.Sprintf(`cd "%s" && exec $SHELL`, dest)
	if len(app.Opts.Params) > 0 {
		// run shell with the command we were given
		script = fmt.Sprintf(
			`%s -c "%s"`, script,
			shellescape.QuoteCommand(app.Opts.Params),
		)
	}

	if err := app.lxcExec(app.sudoLogin(script)...); err != nil {
		return fmt.Errorf("failed to lxc exec: %w", err)
	}
	return nil
}
