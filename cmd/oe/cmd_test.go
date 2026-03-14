package main

import (
	"errors"
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
