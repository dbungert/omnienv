package main

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/stretchr/testify/assert"

	"github.com/dbungert/omnienv/mocks"
	"github.com/stretchr/testify/mock"
)

func mockApp() App {
	return App{Config: Config{Label: "l", System: NewSystem("s")}}
}

func Patch[T any](target *T, mock T) func() {
	original := *target
	*target = mock
	return func() { *target = original }
}

/*
var launchTests = []struct {
	summary     string
	vm          bool
	mockCmds    [][]string
	willCheckVM bool

	runCmds [][]string
	errMsg  string
}{{
	summary:     "simple launch",
	mockCmds:    [][]string{[]string{"true"}, []string{"true"}},
	willCheckVM: true,
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
	errMsg: "",
}, {
	summary:     "launch failed cloud-init",
	mockCmds:    [][]string{[]string{"true"}, []string{"false"}},
	willCheckVM: true,
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
	errMsg: "cloud-init failure: exit status 1",
}, {
	summary:     "launch vm",
	vm:          true,
	mockCmds:    [][]string{[]string{"true"}, []string{"true"}, []string{"true"}},
	willCheckVM: true,
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s", "--vm"},
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
	errMsg: "",
}, {
	summary:     "launch vm failure",
	vm:          true,
	mockCmds:    [][]string{[]string{"false"}},
	willCheckVM: false,
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s", "--vm"},
	},
	errMsg: "failed to create instance: exit status 1",
}, {
	summary:     "vm launch wait strange failure",
	vm:          true,
	mockCmds:    [][]string{[]string{"true"}, []string{"false"}},
	willCheckVM: true,
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s", "--vm"},
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
	},
	errMsg: "failed to wait for instance: strange exit code 1",
}}

func TestLaunch(t *testing.T) {
	for _, test := range launchTests {
		idx := -1
		runCmds := [][]string{}
		restore := Patch(&command, func(arg0 string, rest ...string) *exec.Cmd {
			runCmds = append(runCmds, append([]string{arg0}, rest...))
			idx += 1
			return exec.Command(test.mockCmds[idx][0], test.mockCmds[idx][1:]...)
		})
		defer restore()
		app := mockApp()
		if test.vm {
			app.Config.Virtualization = "vm"
		}
		if test.willCheckVM {
			iType := "container"
			if test.vm {
				iType = "virtual-machine"
			}
			mis := mockGetInstance(t, iType, nil)
			restore := patchConnect(mis, nil)
			defer restore()
		}

		err := app.launch()
		assert.Equal(t, test.runCmds, runCmds, test.summary)

		if len(test.errMsg) > 0 {
			assert.ErrorContains(t, err, test.errMsg, test.summary)
		} else {
			assert.Nil(t, err, test.summary)
		}
	}
}
*/

var waitTests = []struct {
	summary  string
	vm       bool
	mockCmds [][]string

	runCmds [][]string
	err     error
}{{
	summary: "non-vm",
	runCmds: [][]string{},
}, {
	summary: "happy path",
	vm:      true,
	mockCmds: [][]string{
		[]string{"sh", "-c", "exit 255"},
		[]string{"true"},
	},
	runCmds: [][]string{
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
	},
}, {
	summary:  "strange exit",
	vm:       true,
	mockCmds: [][]string{[]string{"sh", "-c", "exit 1"}},
	err:      errors.New("strange exit code 1"),
	runCmds:  [][]string{[]string{"lxc", "exec", "l-s", "--", "/bin/true"}},
}}

func TestWait(t *testing.T) {
	for _, test := range waitTests {
		idx := -1
		runCmds := [][]string{}
		restore := Patch(&command, func(arg0 string, rest ...string) *exec.Cmd {
			runCmds = append(runCmds, append([]string{arg0}, rest...))
			idx += 1
			return exec.Command(test.mockCmds[idx][0], test.mockCmds[idx][1:]...)
		})
		defer restore()
		app := mockApp()
		if test.vm {
			app.Config.Virtualization = "vm"
		}

		iType := "container"
		if test.vm {
			iType = "virtual-machine"
		}
		mis := mockGetInstance(t, iType, nil)
		restoreConnect := patchConnect(mis, nil)
		defer restoreConnect()

		restoreSleep := Patch(&timeSleep, func(d time.Duration) {})
		defer restoreSleep()

		assert.Equal(t, test.err, app.wait(), test.summary)
		assert.Equal(t, test.runCmds, runCmds)
	}
}

func TestStartFailedUIS(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	err := fmt.Errorf("failed start")
	mis.On("UpdateInstanceState", "l-s", mock.Anything, "").Return(nil, err)
	assert.NotNil(t, mockApp().start(mis))
}

func mockUpdateInstanceState(t *testing.T, mis *mocks.MockInstanceServer, err error) {
	op := mocks.NewMockOperation(t)
	mis.On("UpdateInstanceState", "l-s", mock.Anything, "").Return(op, nil)
	op.On("Wait").Return(err)
}

func TestStart(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	mockUpdateInstanceState(t, mis, nil)
	assert.Nil(t, mockApp().start(mis))
}

func TestStartFailedWait(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	mockUpdateInstanceState(t, mis, fmt.Errorf("error"))
	assert.NotNil(t, mockApp().start(mis))
}

func patchConnect(is lxd.InstanceServer, err error) func() {
	restore := Patch(&connectLXDUnix, func(path string, args *lxd.ConnectionArgs) (lxd.InstanceServer, error) {
		return is, err
	})
	return restore
}

func TestStartIfNeeded_ConnectFail(t *testing.T) {
	restore := patchConnect(nil, errors.New("error"))
	defer restore()
	assert.NotNil(t, mockApp().startIfNeeded())
}

func mockGetInstance(t *testing.T, instanceType string, err error) *mocks.MockInstanceServer {
	mis := mocks.NewMockInstanceServer(t)
	instance := api.Instance{Type: instanceType}
	mis.On("GetInstance", "l-s").Return(&instance, "", err)
	mis.On("Disconnect").Return()
	return mis
}

func mockGetInstanceState(t *testing.T, status string, err error) *mocks.MockInstanceServer {
	mis := mocks.NewMockInstanceServer(t)
	state := api.InstanceState{Status: status}
	mis.On("GetInstanceState", "l-s").Return(&state, "", err)
	mis.On("Disconnect").Return()
	return mis
}

func TestStartIfNeeded_GISFail(t *testing.T) {
	mis := mockGetInstanceState(t, "", fmt.Errorf("error"))
	restore := patchConnect(mis, nil)
	defer restore()
	assert.NotNil(t, mockApp().startIfNeeded())
}

func TestStartIfNeeded_UnknownState(t *testing.T) {
	mis := mockGetInstanceState(t, "NotAState", nil)
	restore := patchConnect(mis, nil)
	defer restore()
	assert.NotNil(t, mockApp().startIfNeeded())
}

func TestStartIfNeeded_Running(t *testing.T) {
	mis := mockGetInstanceState(t, "Running", nil)
	restore := patchConnect(mis, nil)
	defer restore()
	assert.Nil(t, mockApp().startIfNeeded())
}

func TestStartIfNeeded_Stopped(t *testing.T) {
	mis := mockGetInstanceState(t, "Stopped", nil)
	mockUpdateInstanceState(t, mis, nil)
	restore := patchConnect(mis, nil)
	defer restore()
	assert.Nil(t, mockApp().startIfNeeded())
}

func TestLxcExec(t *testing.T) {
	restoreLP := Patch(&lookPath, func(file string) (string, error) {
		return "/foo/lxc", nil
	})
	defer restoreLP()

	restoreUser := patchEnv("USER", "user")
	defer restoreUser()

	restoreSE := Patch(&syscallExec, func(argv0 string, argv []string, envv []string) (err error) {
		assert.Equal(t, argv0, "/foo/lxc")
		assert.Equal(t, argv, []string{
			"/foo/lxc", "exec", "-", "--",
			"sudo", "--login", "--user", "user",
			"sh", "-c", "bar",
		})
		return nil
	})
	defer restoreSE()
	assert.Nil(t, App{}.lxcExec("bar"))
}

func TestLxcExecFailedLookup(t *testing.T) {
	err := errors.New("error")
	restoreLP := Patch(&lookPath, func(file string) (string, error) {
		return "", err
	})
	defer restoreLP()
	assert.Equal(t, err, App{}.lxcExec(""))
}

var shellTests = []struct {
	opts   Opts
	script string
}{{
	opts:   Opts{},
	script: `cd "/tmp" && exec $SHELL`,
}, {
	opts:   Opts{Params: []string{"a"}},
	script: `cd "/tmp" && exec $SHELL -c "a"`,
}}

func TestShell(t *testing.T) {
	restorePWD := patchEnv("PWD", "/tmp")
	defer restorePWD()

	restoreUser := patchEnv("USER", "user")
	defer restoreUser()

	restoreLP := Patch(&lookPath, func(file string) (string, error) {
		return "lxc", nil
	})
	defer restoreLP()

	for _, test := range shellTests {
		restoreSE := Patch(&syscallExec, func(argv0 string, argv []string, envv []string) (err error) {
			assert.Equal(t, argv0, "lxc")
			assert.Equal(t, argv, []string{
				"lxc", "exec", "l-s", "--",
				"sudo", "--login", "--user", "user",
				"sh", "-c", test.script,
			})
			return nil
		})
		defer restoreSE()

		mis := mockGetInstanceState(t, "Running", nil)
		restore := patchConnect(mis, nil)
		defer restore()

		instance := api.Instance{Type: "container"}
		mis.On("GetInstance", "l-s").Return(&instance, "", nil)

		app := mockApp()
		app.Opts = test.opts
		assert.Nil(t, app.shell())
	}
}

func TestShell_StartFail(t *testing.T) {
	mis := mockGetInstanceState(t, "", fmt.Errorf("error"))
	restore := patchConnect(mis, nil)
	defer restore()
	assert.NotNil(t, mockApp().shell())
}

func TestShell_LXCFail(t *testing.T) {
	mis := mockGetInstanceState(t, "Running", nil)
	restore := patchConnect(mis, nil)
	defer restore()

	instance := api.Instance{Type: "container"}
	mis.On("GetInstance", "l-s").Return(&instance, "", nil)

	restoreLP := Patch(&lookPath, func(file string) (string, error) {
		return "lxc", nil
	})
	defer restoreLP()

	restoreSE := Patch(&syscallExec, func(argv0 string, argv []string, envv []string) (err error) {
		return errors.New("error")
	})
	defer restoreSE()
	assert.NotNil(t, mockApp().shell())
}
