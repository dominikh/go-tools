//go:build android || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build android darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func fn2() {
	c := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGSTOP)    //@ diag(`cannot be trapped`)
	signal.Notify(c, syscall.SIGSTOP) //@ diag(`cannot be trapped`)
	signal.Reset(syscall.SIGSTOP)     //@ diag(`cannot be trapped`)
}
