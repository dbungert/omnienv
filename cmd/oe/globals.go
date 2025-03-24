package main

import (
	"io"
	"os"
	"os/exec"
	"syscall"

	lxd "github.com/canonical/lxd/client"
)

var command = exec.Command
var connectLXDUnix = lxd.ConnectLXDUnix
var lookPath = exec.LookPath
var exit = os.Exit
var stderr io.Writer = os.Stderr
var syscallExec = syscall.Exec
