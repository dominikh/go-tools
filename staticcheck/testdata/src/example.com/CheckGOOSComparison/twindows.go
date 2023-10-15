//go:build windows

package pkg

import "runtime"

func windows() {
	if runtime.GOOS == "windows" {
	}
	if runtime.GOOS == "linux" { //@ diag(`runtime.GOOS will never equal "linux"`)
	}
}
