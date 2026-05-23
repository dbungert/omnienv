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
