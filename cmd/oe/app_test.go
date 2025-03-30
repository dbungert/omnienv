package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var nameTests = []struct {
	summary string
	config  Config

	name string
}{{
	summary: "basic name",
	config:  Config{Label: "l", System: "s"},
	name:    "l-s",
}, {
	summary: "vm",
	config:  Config{Label: "foo", System: "bar", Virtualization: "vm"},
	name:    "foo-bar",
}}

func TestName(t *testing.T) {
	for _, test := range nameTests {
		app := App{Config: test.config}
		assert.Equal(t, test.name, app.name(), test.summary)
	}
}
