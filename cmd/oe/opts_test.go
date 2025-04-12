package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var argsTests = []struct {
	summary   string
	argsInput []string

	opts Opts
}{{
	summary:   "one arg",
	argsInput: []string{"asdf"},
	opts:      Opts{Params: []string{"asdf"}},
}, {
	summary:   "launch",
	argsInput: []string{"--launch"},
	opts:      Opts{Launch: true},
}, {
	summary:   "verbose",
	argsInput: []string{"--verbose"},
	opts:      Opts{Verbose: true},
}, {
	summary:   "v",
	argsInput: []string{"-v"},
	opts:      Opts{Verbose: true},
}, {
	summary:   "system",
	argsInput: []string{"--system", "foo"},
	opts:      Opts{System: "foo"},
}, {
	summary:   "system + param",
	argsInput: []string{"--system", "foo", "bar"},
	opts:      Opts{System: "foo", Params: []string{"bar"}},
}, {
	summary:   "version",
	argsInput: []string{"--version"},
	opts:      Opts{Version: true},
}, {
	summary:   "pass after unrecognized",
	argsInput: []string{"bash", "--help"},
	opts:      Opts{Params: []string{"bash", "--help"}},
}, {
	summary:   "pass after double dash",
	argsInput: []string{"--", "bash", "--help"},
	opts:      Opts{Params: []string{"bash", "--help"}},
}}

func TestArgs(t *testing.T) {
	for _, test := range argsTests {
		if test.opts.Params == nil {
			// default initializer in the test for Params will be
			// nil, but actual no Params will be an empty array.
			test.opts.Params = []string{}
		}

		opts, err := GetOpts(test.argsInput)
		assert.Nil(t, err, test.summary)
		assert.Equal(t, test.opts, opts, test.summary)
	}
}
