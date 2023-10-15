package pkg

import "runtime"

func fileUnix() {
	// "unix" is not a magic file name suffix, so all of these branches are fine
	if runtime.GOOS == "windows" {
	}
	if runtime.GOOS == "android" {
	}
	if runtime.GOOS == "linux" {
	}
}
