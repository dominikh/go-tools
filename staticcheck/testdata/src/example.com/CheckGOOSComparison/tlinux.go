//go:build linux

package pkg

import "runtime"

func linux() {
	if runtime.GOOS == "windows" { //@ diag(`runtime.GOOS will never equal "windows"`)
	}
	if runtime.GOOS == "android" {
	}
	if runtime.GOOS == "linux" {
	}
}
