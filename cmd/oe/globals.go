package main

import (
	"io"
	"os"
	"os/exec"
	"time"
)

var command = exec.Command
var commandContext = exec.CommandContext
var stderr io.Writer = os.Stderr
var timeSleep = time.Sleep
