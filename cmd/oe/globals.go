package main

import (
	"io"
	"os"
	"os/exec"
	"time"

	lxd "github.com/canonical/lxd/client"
)

var command = exec.Command
var connectLXDUnix = lxd.ConnectLXDUnix
var stderr io.Writer = os.Stderr
var timeSleep = time.Sleep
