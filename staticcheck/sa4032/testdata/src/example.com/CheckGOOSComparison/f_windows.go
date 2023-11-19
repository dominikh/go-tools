package pkg

import "runtime"

func fileWindows() {
	if runtime.GOOS == "windows" {
	}
	if runtime.GOOS == "linux" { //@ diag(`runtime.GOOS will never equal "linux"`)
	}
}
