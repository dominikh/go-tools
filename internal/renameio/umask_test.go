// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !plan9 && !windows && !js
// +build !plan9,!windows,!js

package renameio

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteFileModeAppliesUmask(t *testing.T) {
	dir := t.TempDir()

	const mode = 0644
	const umask = 0007
	defer syscall.Umask(syscall.Umask(umask))

	file := filepath.Join(dir, "testWrite")
	err := WriteFile(file, []byte("go-build"), mode)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	fi, err := os.Stat(file)
	if err != nil {
		t.Fatalf("Stat %q (looking for mode %#o): %s", file, mode, err)
	}

	if fi.Mode()&os.ModePerm != 0640 {
		t.Errorf("Stat %q: mode %#o want %#o", file, fi.Mode()&os.ModePerm, 0640)
	}
}
