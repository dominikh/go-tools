//go:build unix || windows

package pkg

import "runtime"

func unknown() {
	// don't flag this, we don't know what DomOS is.
	if runtime.GOOS == "DomOS" {
	}
}
