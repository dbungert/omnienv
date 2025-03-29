package main

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/stretchr/testify/assert"

	"github.com/dbungert/omnienv/mocks"
	"github.com/stretchr/testify/mock"
)

func Patch[T any](target *T, mock T) func() {
	original := *target
	*target = mock
	return func() { *target = original }
}

var launchTests = []struct {
	config   Config
	mockCmds [][]string

	runCmds [][]string
	errMsg  string
}{{
	config: Config{
		Label:  "l",
		Series: "s",
	},
	mockCmds: [][]string{[]string{"true"}, []string{"true"}},
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
	errMsg: "",
}, {
	config: Config{
		Label:  "l",
		Series: "s",
	},
	mockCmds: [][]string{[]string{"true"}, []string{"false"}},
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
	errMsg: "cloud-init failure: exit status 1",
}, {
	config: Config{
		Label:          "l",
		Series:         "s",
		Virtualization: "vm",
	},
	mockCmds: [][]string{[]string{"true"}, []string{"true"}, []string{"true"}},
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
	config: Config{
		Label:          "l",
		Series:         "s",
		Virtualization: "vm",
	},
	mockCmds: [][]string{
		[]string{"false"},
	},
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s", "--vm"},
	},
	errMsg: "failed to create instance: exit status 1",
}, {
	config: Config{
		Label:          "l",
		Series:         "s",
		Virtualization: "vm",
	},
	mockCmds: [][]string{
		[]string{"true"},
		[]string{"false"},
	},
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
		err := App{Config: test.config}.launch()
		assert.Equal(t, test.runCmds, runCmds)
		if len(test.errMsg) > 0 {
			assert.ErrorContains(t, err, test.errMsg)
		} else {
			assert.Nil(t, err)
		}
	}
}

var waitTests = []struct {
	config   Config
	mockCmds [][]string

	runCmds [][]string
	err     error
}{{
	config: Config{
		Label:  "l",
		Series: "s",
	},
	mockCmds: [][]string{},
	runCmds:  [][]string{},
	err:      nil,
}, {
	config: Config{
		Label:          "l",
		Series:         "s",
		Virtualization: "vm",
	},
	mockCmds: [][]string{
		[]string{"sh", "-c", "exit 255"},
		[]string{"true"},
	},
	runCmds: [][]string{
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
	},
	err: nil,
}, {
	config: Config{
		Label:          "l",
		Series:         "s",
		Virtualization: "vm",
	},
	mockCmds: [][]string{
		[]string{"sh", "-c", "exit 1"},
	},
	runCmds: [][]string{
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
	},
	err: errors.New("strange exit code 1"),
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
		assert.Equal(t, test.err, App{Config: test.config}.wait())
		assert.Equal(t, test.runCmds, runCmds)
	}
}

func TestStartFailedUIS(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	err := fmt.Errorf("failed start")
	mis.On("UpdateInstanceState", "-", mock.Anything, "").Return(nil, err)
	assert.NotNil(t, App{}.start(mis))
}

func mockUpdateInstanceState(t *testing.T, mis *mocks.MockInstanceServer, err error) {
	op := mocks.NewMockOperation(t)
	mis.On("UpdateInstanceState", "-", mock.Anything, "").Return(op, nil)
	op.On("Wait").Return(err)
}

func TestStart(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	mockUpdateInstanceState(t, mis, nil)
	assert.Nil(t, App{}.start(mis))
}

func TestStartFailedWait(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	mockUpdateInstanceState(t, mis, fmt.Errorf("error"))
	assert.NotNil(t, App{}.start(mis))
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
	assert.NotNil(t, App{}.startIfNeeded())
}

func mockGetInstanceState(t *testing.T, status string, err error) *mocks.MockInstanceServer {
	mis := mocks.NewMockInstanceServer(t)
	state := api.InstanceState{Status: status}
	mis.On("GetInstanceState", "-").Return(&state, "", err)
	return mis
}

func TestStartIfNeeded_GISFail(t *testing.T) {
	mis := mockGetInstanceState(t, "", fmt.Errorf("error"))
	restore := patchConnect(mis, nil)
	defer restore()
	assert.NotNil(t, App{}.startIfNeeded())
}

func TestStartIfNeeded_UnknownState(t *testing.T) {
	mis := mockGetInstanceState(t, "NotAState", nil)
	restore := patchConnect(mis, nil)
	defer restore()
	assert.NotNil(t, App{}.startIfNeeded())
}

func TestStartIfNeeded_Running(t *testing.T) {
	mis := mockGetInstanceState(t, "Running", nil)
	restore := patchConnect(mis, nil)
	defer restore()
	assert.Nil(t, App{}.startIfNeeded())
}

func TestStartIfNeeded_Stopped(t *testing.T) {
	mis := mockGetInstanceState(t, "Stopped", nil)
	mockUpdateInstanceState(t, mis, nil)
	restore := patchConnect(mis, nil)
	defer restore()
	assert.Nil(t, App{}.startIfNeeded())
}

func TestLxcExec(t *testing.T) {
	restoreLP := Patch(&lookPath, func(file string) (string, error) {
		return "/foo/lxc", nil
	})
	defer restoreLP()

	restoreSE := Patch(&syscallExec, func(argv0 string, argv []string, envv []string) (err error) {
		assert.Equal(t, argv0, "/foo/lxc")
		assert.Equal(t, argv, []string{
			"/foo/lxc", "exec", "-", "--",
			"su", "-P", "-", "dbungert", "-c", "bar",
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
	opts:   Opts{Params: []string{"a", "b"}},
	script: `cd "/tmp" && exec $SHELL -c "b"`,
}}

func TestShell(t *testing.T) {
	mis := mockGetInstanceState(t, "Running", nil)
	restore := patchConnect(mis, nil)
	defer restore()

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
				"lxc", "exec", "-", "--",
				"su", "-P", "-", "user", "-c",
				test.script,
			})
			return nil
		})
		defer restoreSE()
		assert.Nil(t, App{Config{}, test.opts}.shell())
	}
}

func TestShell_StartFail(t *testing.T) {
	mis := mockGetInstanceState(t, "", fmt.Errorf("error"))
	restore := patchConnect(mis, nil)
	defer restore()
	assert.NotNil(t, App{Config{}, Opts{}}.shell())
}

func TestShell_LXCFail(t *testing.T) {
	mis := mockGetInstanceState(t, "Running", nil)
	restore := patchConnect(mis, nil)
	defer restore()

	restoreLP := Patch(&lookPath, func(file string) (string, error) {
		return "lxc", nil
	})
	defer restoreLP()

	restoreSE := Patch(&syscallExec, func(argv0 string, argv []string, envv []string) (err error) {
		return errors.New("error")
	})
	defer restoreSE()
	assert.NotNil(t, App{Config{}, Opts{}}.shell())
}
