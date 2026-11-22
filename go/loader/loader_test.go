package loader

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func writeModule(t *testing.T, dir, name string) string {
	t.Helper()

	root := filepath.Join(dir, name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.24\n",
		"m.go":   "package m\n\nfunc helper() int { return 1 }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func graphPackage(t *testing.T, dir string) *PackageSpec {
	t.Helper()

	specs, err := Graph(&packages.Config{
		Dir:        dir,
		BuildFlags: []string{"-trimpath"},
	}, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		if spec.ID == "example.com/m" {
			return spec
		}
	}
	t.Fatalf("package example.com/m not found in %s", dir)
	return nil
}

func TestGraphHashIncludesPackageDirectory(t *testing.T) {
	// Under -trimpath, the build ID of a package's export data doesn't
	// include the package's directory. The hash still has to differ for
	// identical packages in different directories, because cached
	// diagnostics refer to files by absolute path.
	root := t.TempDir()
	dirA := writeModule(t, root, "a")
	dirB := writeModule(t, root, "b")

	specA := graphPackage(t, dirA)
	specB := graphPackage(t, dirB)
	if specA.Dir == specB.Dir {
		t.Fatalf("package directories are equal: %q", specA.Dir)
	}

	idA, err := getBuildid(specA.ExportFile)
	if err != nil {
		t.Fatal(err)
	}
	idB, err := getBuildid(specB.ExportFile)
	if err != nil {
		t.Fatal(err)
	}
	if idA != idB {
		t.Skipf("build IDs differ (%q and %q); this test requires -trimpath to omit the package directory", idA, idB)
	}
	if specA.Hash == specB.Hash {
		t.Fatalf("package hashes are equal for %q and %q", specA.Dir, specB.Dir)
	}
}
