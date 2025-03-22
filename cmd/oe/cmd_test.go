package main

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
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

var launchTest = []struct {
	config Config

	cmds [][]string
}{{
	config: Config{
		Label:  "l",
		Series: "s",
	},
	cmds: [][]string{
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
	cmds: [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s", "--vm"},
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	},
}}

func TestLaunch(t *testing.T) {
	var cmds [][]string
	restore := Patch(&command, func(arg0 string, rest ...string) *exec.Cmd {
		cmds = append(cmds, append([]string{arg0}, rest...))
		return exec.Command("/bin/true")
	})
	defer restore()

	for _, test := range launchTest {
		cmds = [][]string{}
		launch(test.config)
		assert.Equal(t, test.cmds, cmds)
	}
}
