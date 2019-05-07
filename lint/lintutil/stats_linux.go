package lintutil

import (
	"os"
	"syscall"
)

var infoSignals = []os.Signal{syscall.SIGUSR1}
