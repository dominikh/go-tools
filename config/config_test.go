package config

import (
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	// Layout used by the tests:
	//
	//	root/
	//		cache/           (stands in for os.UserCacheDir)
	//			go-build/    (the build cache, GOCACHE)
	//			repro/       (a user package below the user cache dir)
	//		cache-like/      (shares a string prefix with cache/)
	root := filepath.Join(string(filepath.Separator), "root")
	cache := filepath.Join(root, "cache")
	buildCache := filepath.Join(cache, "go-build")
	repro := filepath.Join(cache, "repro")

	tests := []struct {
		name       string
		files      []string
		buildCache string
		want       string
	}{
		{
			name:       "no files",
			files:      nil,
			buildCache: buildCache,
			want:       "",
		},
		{
			name:       "regular package",
			files:      []string{filepath.Join(root, "src", "x.go")},
			buildCache: buildCache,
			want:       filepath.Join(root, "src"),
		},
		{
			// Issue #1847: a package below the user cache directory
			// must not be mistaken for a package in the build cache.
			name:       "package below user cache dir",
			files:      []string{filepath.Join(repro, "repro.go")},
			buildCache: buildCache,
			want:       repro,
		},
		{
			name:       "package sharing prefix with user cache dir",
			files:      []string{filepath.Join(root, "cache-like", "repro", "repro.go")},
			buildCache: buildCache,
			want:       filepath.Join(root, "cache-like", "repro"),
		},
		{
			name:       "package sharing prefix with build cache",
			files:      []string{filepath.Join(cache, "go-build-like", "repro.go")},
			buildCache: buildCache,
			want:       filepath.Join(cache, "go-build-like"),
		},
		{
			name:       "package only in build cache",
			files:      []string{filepath.Join(buildCache, "ab", "ab123", "_testmain.go")},
			buildCache: buildCache,
			want:       "",
		},
		{
			name:       "file directly in build cache",
			files:      []string{filepath.Join(buildCache, "x.go")},
			buildCache: buildCache,
			want:       "",
		},
		{
			name: "first file in build cache, second is real",
			files: []string{
				filepath.Join(buildCache, "ab", "ab123", "_testmain.go"),
				filepath.Join(repro, "repro.go"),
			},
			buildCache: buildCache,
			want:       repro,
		},
		{
			name:       "unknown build cache disables skipping",
			files:      []string{filepath.Join(buildCache, "ab", "x.go")},
			buildCache: "",
			want:       filepath.Join(buildCache, "ab"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dir(tt.files, tt.buildCache); got != tt.want {
				t.Errorf("dir(%q, %q) = %q, want %q", tt.files, tt.buildCache, got, tt.want)
			}
		})
	}
}

func TestDirPublic(t *testing.T) {
	// A package in a regular directory gets its directory, exercising
	// the real build cache lookup behind Dir.
	d := t.TempDir()
	f := filepath.Join(d, "x.go")
	if got := Dir([]string{f}); got != d {
		t.Errorf("Dir(%q) = %q, want %q", f, got, d)
	}
	if got := Dir(nil); got != "" {
		t.Errorf("Dir(nil) = %q, want %q", got, "")
	}
}
