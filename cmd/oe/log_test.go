package main

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var logTests = []struct {
	summary string
	verbose bool
}{{
	summary: "verbose false",
	verbose: false,
}, {
	summary: "verbose true",
	verbose: true,
}}

func patchLogger() (func(), *bytes.Buffer) {
	buf := &bytes.Buffer{}
	var mockStderr io.Writer = buf
	return Patch(&stderr, mockStderr), buf
}

func TestSetupLogging(t *testing.T) {
	restore, buf := patchLogger()
	defer restore()

	for _, test := range logTests {
		buf.Reset()
		setupLogging(Opts{Verbose: test.verbose})

		slog.Info("info")
		slog.Debug("debug")

		lines := strings.Split(buf.String(), "\n")
		assert.Contains(t, lines[0], "level=INFO msg=info")
		if test.verbose {
			assert.Contains(t, lines[1], "level=DEBUG msg=debug")
			assert.Len(t, lines, 3)
		} else {
			assert.Len(t, lines, 2)
		}
	}
}
