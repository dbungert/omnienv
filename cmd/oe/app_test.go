package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var nameTests = []struct {
	summary string
	config  Config
	opts    Opts

	name string
}{{
	summary: "basic name",
	config:  Config{Label: "l", System: "s"},
	name:    "l-s",
}, {
	summary: "foo-bar",
	config:  Config{Label: "foo", System: "bar"},
	name:    "foo-bar",
}, {
	summary: "opts override",
	config:  Config{Label: "l", System: "sys-from-config"},
	opts:    Opts{System: "sys-from-opts"},
	name:    "l-sys-from-opts",
}}

func TestName(t *testing.T) {
	for _, test := range nameTests {
		app := App{Config: test.config, Opts: test.opts}
		assert.Equal(t, test.name, app.name(), test.summary)
	}
}
