//go:build linux || windows

package pkg

import "runtime"

func complex() {
	if runtime.GOOS == "linux" {
	}
	if runtime.GOOS == "android" {
	}
	if runtime.GOOS == "windows" {
	}
	if runtime.GOOS == "darwin" { //@ diag(`runtime.GOOS will never equal "darwin"`)
	}
}
