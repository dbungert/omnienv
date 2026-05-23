package main

import (
	"io"
	"os"
	"os/exec"
	"time"
)

var command = exec.Command
var stderr io.Writer = os.Stderr
var timeSleep = time.Sleep
