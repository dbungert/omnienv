package main

import (
	"errors"
	"os/exec"
	"testing"

	lxd "github.com/canonical/lxd/client"
	"github.com/stretchr/testify/assert"
)

func mockApp() App {
	return App{Config: Config{Label: "l", System: NewSystem("s")}}
}

func Patch[T any](target *T, mock T) func() {
	original := *target
	*target = mock
	return func() { *target = original }
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
