package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var argsTests = []struct {
	summary   string
	argsInput []string

	opts Opts
	err  error
}{{
	summary:   "one arg",
	argsInput: []string{"asdf"},
	opts:      Opts{Params: []string{"asdf"}},
	err:       nil,
}, {
	summary:   "launch",
	argsInput: []string{"--launch"},
	opts:      Opts{Launch: true},
	err:       nil,
}, {
	summary:   "verbose",
	argsInput: []string{"--verbose"},
	opts:      Opts{Verbose: true},
	err:       nil,
}, {
	summary:   "v",
	argsInput: []string{"-v"},
	opts:      Opts{Verbose: true},
	err:       nil,
}, {
	summary:   "system",
	argsInput: []string{"--system", "foo"},
	opts:      Opts{System: "foo"},
	err:       nil,
}, {
	summary:   "system + param",
	argsInput: []string{"--system", "foo", "bar"},
	opts:      Opts{System: "foo", Params: []string{"bar"}},
	err:       nil,
}}

func TestArgs(t *testing.T) {
	for _, test := range argsTests {
		if test.opts.Params == nil {
			// default initializer in the test for Params will be
			// nil, but actual no Params will be an empty array.
			test.opts.Params = []string{}
		}

		opts, err := GetOpts(test.argsInput)
		assert.Equal(t, test.err, err, test.summary)
		assert.Equal(t, test.opts, opts, test.summary)
	}
}
