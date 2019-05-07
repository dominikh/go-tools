// +build !darwin,!dragonfly,!freebsd,!netbsd,!openbsd,!linux

package lintutil

import "os"

var infoSignals = []os.Signal{}
