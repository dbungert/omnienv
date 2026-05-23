package omnienv

import (
	"os/exec"
	"time"
)

var command = exec.Command
var commandContext = exec.CommandContext
var timeSleep = time.Sleep
