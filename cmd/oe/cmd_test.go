package main

import (
	"fmt"
	"os/exec"
	"testing"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/stretchr/testify/assert"

	"github.com/dbungert/omnienv/mocks"
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

type mockInstanceServer struct {
	op lxd.Operation
}

func (mis mockInstanceServer) UpdateInstanceState(name string, state api.InstanceStatePut, ETag string) (op lxd.Operation, err error) {
	return mis.op, nil
}

func TestStart(t *testing.T) {
	mockOp := mocks.NewMockOperation(t)
	mockOp.On("Wait").Return(nil)
	assert.Nil(t, start(mockInstanceServer{mockOp}, Config{}))
}

func TestStartFailedWait(t *testing.T) {
	mockOp := mocks.NewMockOperation(t)
	mockOp.On("Wait").Return(fmt.Errorf("disaster"))
	assert.NotNil(t, start(mockInstanceServer{mockOp}, Config{}))
}
