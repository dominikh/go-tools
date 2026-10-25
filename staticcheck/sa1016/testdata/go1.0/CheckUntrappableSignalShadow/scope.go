package pkg

import (
	"os"
	"os/signal"
	sys "syscall"
)

var _ = sys.SIGTERM

func fn(c chan os.Signal) {
	sys := 1
	_ = sys
	signal.Notify(c, os.Kill) //@ diag(`cannot be trapped`)
}
