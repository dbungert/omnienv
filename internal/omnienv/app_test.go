package omnienv

import (
	"os/exec"
	"testing"

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
