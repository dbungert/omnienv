package main

import (
	"testing"

	"github.com/dbungert/omnienv/internal/omnienv"
	"github.com/stretchr/testify/assert"
)

var argsTests = []struct {
	summary   string
	argsInput []string

	opts omnienv.Opts
}{{
	summary:   "one arg",
	argsInput: []string{"asdf"},
	opts:      omnienv.Opts{Params: []string{"asdf"}},
}, {
	summary:   "launch",
	argsInput: []string{"--launch"},
	opts:      omnienv.Opts{Launch: true},
}, {
	summary:   "verbose",
	argsInput: []string{"--verbose"},
	opts:      omnienv.Opts{Verbose: true},
}, {
	summary:   "v",
	argsInput: []string{"-v"},
	opts:      omnienv.Opts{Verbose: true},
}, {
	summary:   "system",
	argsInput: []string{"--system", "foo"},
	opts:      omnienv.Opts{System: "foo"},
}, {
	summary:   "system + param",
	argsInput: []string{"--system", "foo", "bar"},
	opts:      omnienv.Opts{System: "foo", Params: []string{"bar"}},
}, {
	summary:   "version",
	argsInput: []string{"--version"},
	opts:      omnienv.Opts{Version: true},
}, {
	summary:   "pass after unrecognized",
	argsInput: []string{"bash", "--help"},
	opts:      omnienv.Opts{Params: []string{"bash", "--help"}},
}, {
	summary:   "pass after double dash",
	argsInput: []string{"--", "bash", "--help"},
	opts:      omnienv.Opts{Params: []string{"bash", "--help"}},
}}

func TestArgs(t *testing.T) {
	for _, test := range argsTests {
		if test.opts.Params == nil {
			test.opts.Params = []string{}
		}

		opts, err := GetOpts(test.argsInput)
		assert.Nil(t, err, test.summary)
		assert.Equal(t, test.opts, opts, test.summary)
	}
}

func TestBadArgs(t *testing.T) {
	_, err := GetOpts([]string{"--invalid"})
	assert.NotNil(t, err)
}
