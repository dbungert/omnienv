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

func TestLaunch(t *testing.T) {
	var args [][]string
	restore := Patch(&command, func(arg0 string, rest ...string) *exec.Cmd {
		args = append(args, append([]string{arg0}, rest...))
		return exec.Command("/bin/true")
	})
	defer restore()

	launch(Config{
		Label:  "l",
		Series: "s",
	})
	expected := [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	}
	assert.Equal(t, expected, args)
}

func TestLaunchVM(t *testing.T) {
	var args [][]string
	restore := Patch(&command, func(arg0 string, rest ...string) *exec.Cmd {
		args = append(args, append([]string{arg0}, rest...))
		return exec.Command("/bin/true")
	})
	defer restore()

	launch(Config{
		Label:          "l",
		Series:         "s",
		Virtualization: "vm",
	})
	expected := [][]string{
		[]string{"lxc", "launch", "ubuntu-daily:s", "l-s", "--vm"},
		[]string{"lxc", "exec", "l-s", "--", "/bin/true"},
		[]string{
			"lxc", "exec", "l-s", "--",
			"cloud-init", "status", "--wait",
		},
	}
	assert.Equal(t, expected, args)
}
