package pkg

import (
	"os"
	"path/filepath"
)

func fn1() {
	x := os.TempDir()
	defer os.RemoveAll(x) //@ diag(`deletes the user's entire temporary directory`)

	x = ""
}

func fn2() {
	x := os.TempDir()
	if true {
		os.RemoveAll(x) //@ diag(`deletes the user's entire temporary directory`)
	}
}

func fn3() {
	x := os.TempDir()
	if true {
		x = filepath.Join(x, "foo")
	}
	// we don't flag this to avoid false positives in infeasible paths
	os.RemoveAll(x)
}

func fn4() {
	x, _ := os.UserCacheDir()
	os.RemoveAll(x) //@ diag(`deletes the user's entire cache directory`)

	x, _ = os.UserConfigDir()
	os.RemoveAll(x) //@ diag(`deletes the user's entire config directory`)

	x, _ = os.UserHomeDir()
	os.RemoveAll(x) //@ diag(`deletes the user's entire home directory`)
}
