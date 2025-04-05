package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

var suCanPtyTests = []struct {
	system     string
	optsSystem string
	result     bool
}{
	{"plucky", "", true},
	{"noble", "", true},
	{"jammy", "", true},
	{"jammy", "bionic", false},
	{"focal", "", false},
	{"bionic", "", false},
	{"bionic", "jammy", true},
	{"25.04", "", true},
	{"24.04", "", true},
	{"22.04", "", true},
	{"20.04", "", false},
	{"18.04", "", false},
}

func TestCanUseSuPty(t *testing.T) {
	for _, test := range suCanPtyTests {
		app := App{
			Config{System: NewSystem(test.system)},
			Opts{System: test.optsSystem},
		}
		assert.Equal(t, test.result, app.suCanPty())
	}
}
