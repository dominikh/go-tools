package cache

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDefaultRejectsRelativeCachePathWithoutCreatingIt(t *testing.T) {
	oldDirOnce, oldDir, oldDirErr := defaultDirOnce, defaultDir, defaultDirErr
	oldOnce, oldCache := defaultOnce, defaultCache
	t.Cleanup(func() {
		defaultDirOnce, defaultDir, defaultDirErr = oldDirOnce, oldDir, oldDirErr
		defaultOnce, defaultCache = oldOnce, oldCache
	})
	defaultDirOnce = sync.Once{}
	defaultDir = ""
	defaultDirErr = nil
	defaultOnce = sync.Once{}
	defaultCache = nil

	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	t.Cleanup(func() {
		if err := os.Chdir(oldWorkingDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STATICCHECK_CACHE", "relative-cache")

	if got, err := Default(); err == nil || got != nil {
		t.Fatalf("Default() = (%v, %v), want (nil, error)", got, err)
	}
	if _, err := os.Stat(filepath.Join("relative-cache")); !os.IsNotExist(err) {
		t.Fatalf("relative cache path was created: stat error = %v", err)
	}
}
