//go:build !(linux || windows || unix)

package pkg

import "runtime"

func other() {
	if runtime.GOOS == "linux" { //@ diag(`runtime.GOOS will never equal "linux"`)
	}
}
