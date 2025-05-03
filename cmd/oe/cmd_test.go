package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Patch[T any](target *T, mock T) func() {
	original := *target
	*target = mock
	return func() { *target = original }
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
