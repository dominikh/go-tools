package pkg

import (
	"os"
	"os/signal"
	system "syscall"
)

func fn(c chan os.Signal) {
	signal.Notify(c, system.SIGKILL) //@ diag(`cannot be trapped`)
}
