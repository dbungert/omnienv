package omnienv

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var nameTests = []struct {
	summary string
	config  Config
	opts    Opts

	system      string
	name        string
	launchImage string
}{{
	summary:     "basic name",
	config:      Config{Label: "l", System: NewSystem("s")},
	system:      "s",
	name:        "l-s",
	launchImage: "ubuntu-daily:s",
}, {
	summary:     "foo-bar",
	config:      Config{Label: "foo", System: NewSystem("bar")},
	system:      "bar",
	name:        "foo-bar",
	launchImage: "ubuntu-daily:bar",
}, {
	summary:     "opts override",
	config:      Config{Label: "l", System: NewSystem("sys-from-config")},
	opts:        Opts{System: "sys-from-opts"},
	system:      "sys-from-opts",
	name:        "l-sys-from-opts",
	launchImage: "ubuntu-daily:sys-from-opts",
}}

func TestName(t *testing.T) {
	for _, test := range nameTests {
		app := App{Config: test.config, Opts: test.opts}
		assert.Equal(t, test.system, app.system(), test.summary)
		assert.Equal(t, test.name, app.name(), test.summary)
		assert.Equal(t, test.launchImage, app.launchImage(), test.summary)
	}
}

func TestStartIfNeededRunning(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Status: RUNNING")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.StartIfNeeded())
}

func TestStartIfNeededStopped(t *testing.T) {
	callCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Status: STOPPED")
		}
		return exec.Command("/bin/true")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.StartIfNeeded())
}

func TestStartIfNeededUnknownStatus(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Status: UNKNOWN")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.StartIfNeeded()
	assert.ErrorContains(t, err, "no handler for Status UNKNOWN")
}

func TestStartIfNeededInfoFails(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/false")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.StartIfNeeded()
	assert.ErrorContains(t, err, "failed to get instance info")
}

func TestStartIfNeededNoStatus(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "just some output")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.StartIfNeeded()
	assert.ErrorContains(t, err, "could not determine status")
}

func TestIsVMVM(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Type: virtual-machine")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	vm, err := app.isVM()
	assert.Nil(t, err)
	assert.True(t, vm)
}

func TestIsVMContainer(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Type: container")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	vm, err := app.isVM()
	assert.Nil(t, err)
	assert.False(t, vm)
}

func TestIsVMInfoFails(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/false")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	_, err := app.isVM()
	assert.ErrorContains(t, err, "failed to get instance info")
}

func TestIsVMNoType(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Status: RUNNING")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	_, err := app.isVM()
	assert.ErrorContains(t, err, "could not determine type")
}

func TestWaitNotVM(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Type: container")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.Wait())
}

func TestWaitVMExecOk(t *testing.T) {
	callCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Type: virtual-machine")
		}
		return exec.Command("/bin/true")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.Wait())
}

func TestWaitVMExecFailsNonExitError(t *testing.T) {
	callCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Type: virtual-machine")
		}
		return exec.Command("/nonexistent-binary")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Wait()
	assert.Error(t, err)
}

func TestWaitVMStrangeExitCode(t *testing.T) {
	callCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Type: virtual-machine")
		}
		return exec.Command("/bin/false")
	})
	defer restoreCmd()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Wait()
	assert.ErrorContains(t, err, "strange exit code 1")
}

func TestWaitVMTimeout(t *testing.T) {
	callCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Type: virtual-machine")
		}
		return exec.Command("/bin/sh", "-c", "exit 255")
	})
	defer restoreCmd()
	restoreSleep := Patch(&timeSleep, func(_ time.Duration) {})
	defer restoreSleep()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Wait()
	assert.ErrorContains(t, err, "timed out waiting")
}

func TestWaitVMEventualSuccess(t *testing.T) {
	callCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Type: virtual-machine")
		}
		if callCount < 5 {
			return exec.Command("/bin/sh", "-c", "exit 255")
		}
		return exec.Command("/bin/true")
	})
	defer restoreCmd()
	restoreSleep := Patch(&timeSleep, func(_ time.Duration) {})
	defer restoreSleep()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.Wait())
}

func TestIsUbuntuJammyTrue(t *testing.T) {
	callCount := 0
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Ubuntu")
		}
		return exec.Command("/bin/echo", "22.04")
	})
	defer restoreCmdCtx()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	jammy, err := app.isUbuntuJammy()
	assert.Nil(t, err)
	assert.True(t, jammy)
}

func TestLaunchContainerOk(t *testing.T) {
	cmdCallCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		cmdCallCount++
		switch cmdCallCount {
		case 1:
			return exec.Command("/bin/true") // lxc launch
		case 2:
			return exec.Command("/bin/echo", "Type: container") // lxc info
		case 3:
			return exec.Command("/bin/true") // lxc exec use_pty
		case 4:
			return exec.Command("/bin/true") // lxc exec cloud-init
		default:
			return exec.Command("/bin/true")
		}
	})
	defer restoreCmd()

	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Debian")
	})
	defer restoreCmdCtx()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.Launch())
}

func TestLaunchLaunchFails(t *testing.T) {
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/false")
	})
	defer restoreCmd()

	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Debian")
	})
	defer restoreCmdCtx()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Launch()
	assert.ErrorContains(t, err, "failed to create instance")
}

func TestLaunchWaitFails(t *testing.T) {
	cmdCallCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		cmdCallCount++
		if cmdCallCount == 1 {
			return exec.Command("/bin/true") // lxc launch
		}
		return exec.Command("/bin/false") // lxc info
	})
	defer restoreCmd()

	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Debian")
	})
	defer restoreCmdCtx()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Launch()
	assert.ErrorContains(t, err, "failed to wait for instance")
}

func TestLaunchUsePtyFails(t *testing.T) {
	cmdCallCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		cmdCallCount++
		switch cmdCallCount {
		case 1:
			return exec.Command("/bin/true") // lxc launch
		case 2:
			return exec.Command("/bin/echo", "Type: container") // lxc info
		case 3:
			return exec.Command("/bin/false") // lxc exec use_pty
		default:
			return exec.Command("/bin/true")
		}
	})
	defer restoreCmd()

	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Debian")
	})
	defer restoreCmdCtx()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Launch()
	assert.ErrorContains(t, err, "use_pty setup failure")
}

func TestLaunchCloudInitFails(t *testing.T) {
	cmdCallCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		cmdCallCount++
		switch cmdCallCount {
		case 1:
			return exec.Command("/bin/true") // lxc launch
		case 2:
			return exec.Command("/bin/echo", "Type: container") // lxc info
		case 3:
			return exec.Command("/bin/true") // lxc exec use_pty
		case 4:
			return exec.Command("/bin/false") // lxc exec cloud-init
		default:
			return exec.Command("/bin/true")
		}
	})
	defer restoreCmd()

	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Debian")
	})
	defer restoreCmdCtx()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Launch()
	assert.ErrorContains(t, err, "cloud-init failure")
}

func TestLaunchQuirkFails(t *testing.T) {
	ctxCallCount := 0
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		ctxCallCount++
		if ctxCallCount == 1 {
			return exec.Command("/bin/echo", "Ubuntu")
		}
		return exec.Command("/bin/echo", "22.04")
	})
	defer restoreCmdCtx()

	cmdCallCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		cmdCallCount++
		switch cmdCallCount {
		case 1:
			return exec.Command("/bin/true") // lxc launch
		case 2:
			return exec.Command("/bin/echo", "Type: container") // lxc info
		case 3:
			return exec.Command("/bin/true") // lxc exec use_pty
		case 4:
			return exec.Command("/bin/false") // lxc exec bus wait (quirk)
		default:
			return exec.Command("/bin/true")
		}
	})
	defer restoreCmd()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.Launch()
	assert.ErrorContains(t, err, "LP #1878225 workaround failure")
}

func TestLp1878225QuirkNotJammy(t *testing.T) {
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Debian")
	})
	defer restoreCmdCtx()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.lp1878225Quirk())
}

func TestLp1878225QuirkBusWaitFails(t *testing.T) {
	ctxCallCount := 0
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		ctxCallCount++
		if ctxCallCount == 1 {
			return exec.Command("/bin/echo", "Ubuntu")
		}
		return exec.Command("/bin/echo", "22.04")
	})
	defer restoreCmdCtx()

	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/false")
	})
	defer restoreCmd()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.lp1878225Quirk()
	assert.ErrorContains(t, err, "bus wait failure")
}

func TestLp1878225QuirkSeededStopFails(t *testing.T) {
	ctxCallCount := 0
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		ctxCallCount++
		if ctxCallCount == 1 {
			return exec.Command("/bin/echo", "Ubuntu")
		}
		return exec.Command("/bin/echo", "22.04")
	})
	defer restoreCmdCtx()

	cmdCallCount := 0
	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		cmdCallCount++
		if cmdCallCount == 1 {
			return exec.Command("/bin/true")
		}
		return exec.Command("/bin/false")
	})
	defer restoreCmd()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	err := app.lp1878225Quirk()
	assert.ErrorContains(t, err, "seeded stop failure")
}

func TestLp1878225QuirkJammyOk(t *testing.T) {
	ctxCallCount := 0
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		ctxCallCount++
		if ctxCallCount == 1 {
			return exec.Command("/bin/echo", "Ubuntu")
		}
		return exec.Command("/bin/echo", "22.04")
	})
	defer restoreCmdCtx()

	restoreCmd := Patch(&command, func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/true")
	})
	defer restoreCmd()

	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	assert.Nil(t, app.lp1878225Quirk())
}

func TestIsUbuntuJammyNotUbuntu(t *testing.T) {
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/echo", "Debian")
	})
	defer restoreCmdCtx()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	jammy, err := app.isUbuntuJammy()
	assert.Nil(t, err)
	assert.False(t, jammy)
}

func TestIsUbuntuJammyFirstFails(t *testing.T) {
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("/bin/false")
	})
	defer restoreCmdCtx()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	jammy, err := app.isUbuntuJammy()
	assert.Nil(t, err)
	assert.False(t, jammy)
}

func TestIsUbuntuJammyWrongVersion(t *testing.T) {
	callCount := 0
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Ubuntu")
		}
		return exec.Command("/bin/echo", "24.04")
	})
	defer restoreCmdCtx()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	jammy, err := app.isUbuntuJammy()
	assert.Nil(t, err)
	assert.False(t, jammy)
}

func TestIsUbuntuJammyVersionFails(t *testing.T) {
	callCount := 0
	restoreCmdCtx := Patch(&commandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			return exec.Command("/bin/echo", "Ubuntu")
		}
		return exec.Command("/bin/false")
	})
	defer restoreCmdCtx()
	app := App{Config: Config{Label: "l", System: NewSystem("s")}}
	jammy, err := app.isUbuntuJammy()
	assert.Nil(t, err)
	assert.False(t, jammy)
}
