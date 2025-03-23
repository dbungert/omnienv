package main

import (
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

func TestCheck(t *testing.T) {
	check("/bin/true")
}

func TestCheckBad(t *testing.T) {
	var code int
	restore := Patch(&exit, func(_code int) {
		code = _code
	})
	defer restore()
	check("/bin/false")
	assert.Equal(t, 1, code)
}

var launchTests = []struct {
	config Config

	runCmds [][]string
}{{
	config: Config{
		Label:  "l",
		Series: "s",
	},
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
}, {
	config: Config{
		Label:          "l",
		Series:         "s",
		Virtualization: "vm",
	},
	runCmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s", "--vm"},
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
}}

func TestLaunch(t *testing.T) {
	var runCmds [][]string
	restore := Patch(&command, func(arg0 string, rest ...string) *exec.Cmd {
		runCmds = append(runCmds, append([]string{arg0}, rest...))
		return exec.Command("true")
	})
	defer restore()

	for _, test := range launchTests {
		runCmds = [][]string{}
		launch(test.config)
		assert.Equal(t, test.runCmds, runCmds)
	}
}

var waitTests = []struct {
	config   Config
	mockCmds [][]string

	runCmds [][]string
}{{
	config: Config{
		Label:  "l",
		Series: "s",
	},
	mockCmds: [][]string{},
	runCmds:  [][]string{},
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
		wait(test.config)
		assert.Equal(t, test.runCmds, runCmds)
	}
}

func TestStartFailedUIS(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	err := fmt.Errorf("failed start")
	mis.On("UpdateInstanceState", "-", mock.Anything, "").Return(nil, err)
	assert.NotNil(t, start(mis, Config{}))
}

func TestStart(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	op := mocks.NewMockOperation(t)
	mis.On("UpdateInstanceState", "-", mock.Anything, "").Return(op, nil)
	op.On("Wait").Return(nil)
	assert.Nil(t, start(mis, Config{}))
}

func TestStartFailedWait(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	op := mocks.NewMockOperation(t)
	mis.On("UpdateInstanceState", "-", mock.Anything, "").Return(op, nil)
	op.On("Wait").Return(fmt.Errorf("disaster"))
	assert.NotNil(t, start(mis, Config{}))
}

func TestStartIfNeeded_ConnectFail(t *testing.T) {
	restore := Patch(&connectLXDUnix, func(path string, args *lxd.ConnectionArgs) (lxd.InstanceServer, error) {
		return nil, fmt.Errorf("oh no")
	})
	defer restore()
	assert.NotNil(t, startIfNeeded(Config{}))
}

func TestStartIfNeeded_Running(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	state := api.InstanceState{Status: "Running"}
	mis.On("GetInstanceState", "-").Return(&state, "", nil)
	restore := Patch(&connectLXDUnix, func(path string, args *lxd.ConnectionArgs) (lxd.InstanceServer, error) {
		return mis, nil
	})
	defer restore()
	assert.Nil(t, startIfNeeded(Config{}))
}

func TestStartIfNeeded_Stopped(t *testing.T) {
	mis := mocks.NewMockInstanceServer(t)
	state := api.InstanceState{Status: "Stopped"}
	mis.On("GetInstanceState", "-").Return(&state, "", nil)
	op := mocks.NewMockOperation(t)
	mis.On("UpdateInstanceState", "-", mock.Anything, "").Return(op, nil)
	op.On("Wait").Return(nil)
	restore := Patch(&connectLXDUnix, func(path string, args *lxd.ConnectionArgs) (lxd.InstanceServer, error) {
		return mis, nil
	})
	defer restore()
	assert.Nil(t, startIfNeeded(Config{}))
}
