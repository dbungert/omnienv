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

var nameTests = []struct {
	summary string
	config  Config
	opts    Opts

	system      string
	name        string
	launchImage string
}{{
	summary:     "basic name",
	config:      Config{Label: "l", System: NewSystem("s")},
	system:      "s",
	name:        "l-s",
	launchImage: "ubuntu-daily:s",
}, {
	summary:     "foo-bar",
	config:      Config{Label: "foo", System: NewSystem("bar")},
	system:      "bar",
	name:        "foo-bar",
	launchImage: "ubuntu-daily:bar",
}, {
	summary:     "opts override",
	config:      Config{Label: "l", System: NewSystem("sys-from-config")},
	opts:        Opts{System: "sys-from-opts"},
	system:      "sys-from-opts",
	name:        "l-sys-from-opts",
	launchImage: "ubuntu-daily:sys-from-opts",
}}

func TestName(t *testing.T) {
	for _, test := range nameTests {
		app := App{Config: test.config, Opts: test.opts}
		assert.Equal(t, test.system, app.system(), test.summary)
		assert.Equal(t, test.name, app.name(), test.summary)
		assert.Equal(t, test.launchImage, app.launchImage(), test.summary)
	}
}
