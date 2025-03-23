package main

import (
	"io"
	"os"
	"os/exec"

	lxd "github.com/canonical/lxd/client"
)

var command = exec.Command
var connectLXDUnix = lxd.ConnectLXDUnix
var exit = os.Exit
var stderr io.Writer = os.Stderr
