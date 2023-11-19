//go:build unix

package pkg

import "runtime"

func unix() {
	if runtime.GOOS == "windows" { //@ diag(`runtime.GOOS will never equal "windows"`)
	}
	if runtime.GOOS == "linux" {
	}
	if runtime.GOOS == "darwin" {
	}
}
