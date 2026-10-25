package pkg

import (
	"os"
	"os/signal"
)

func fn2(c chan os.Signal) {
	signal.Notify(c, os.Kill) //@ diag(`cannot be trapped`)
}
