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

func TestLxcExec(t *testing.T) {
	restoreCmd := Patch(&command, func(arg0 string, argv ...string) *exec.Cmd {
		assert.Equal(t, "lxc", arg0)
		assert.Equal(t, []string{"exec", "-", "--", "bar"}, argv)
		cmd := exec.Command("/bin/true")
		cmd.Args = append([]string{arg0}, argv...)
		return cmd
	})
	defer restoreCmd()
	assert.Nil(t, App{}.lxcExec("bar"))
}
