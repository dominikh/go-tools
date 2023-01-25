package main

import (
	"os"
	"os/signal"
	"syscall"
)

func fn() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Ignore(os.Signal(syscall.SIGKILL)) //@ diag(`cannot be trapped`)
	signal.Ignore(os.Kill)                    //@ diag(`cannot be trapped`)
	signal.Notify(c, os.Kill)                 //@ diag(`cannot be trapped`)
	signal.Reset(os.Kill)                     //@ diag(`cannot be trapped`)
	signal.Ignore(syscall.SIGKILL)            //@ diag(`cannot be trapped`)
	signal.Notify(c, syscall.SIGKILL)         //@ diag(`cannot be trapped`)
	signal.Reset(syscall.SIGKILL)             //@ diag(`cannot be trapped`)
}
