package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

const Version = "devel"

// version returns a version descriptor and reports whether the
// version is a known release.
func version() (string, bool) {
	if Version != "devel" {
		return Version, true
	}
	v, ok := buildInfoVersion()
	if ok {
		return v, false
	}
	return "devel", false
}

func Print() {
	v, release := version()

	if release {
		fmt.Printf("%s %s\n", filepath.Base(os.Args[0]), v)
	} else if v == "devel" {
		fmt.Printf("%s (no version)\n", filepath.Base(os.Args[0]))
	} else {
		fmt.Printf("%s (devel, %s)\n", filepath.Base(os.Args[0]), v)
	}
}

func Verbose() {
	Print()
	fmt.Println()
	fmt.Println("Compiled with Go version:", runtime.Version())
	printBuildInfo()
}

func printModule(m *debug.Module) {
	fmt.Printf("\t%s", m.Path)
	if m.Version != "(devel)" {
		fmt.Printf("@%s", m.Version)
	}
	if m.Sum != "" {
		fmt.Printf(" (sum: %s)", m.Sum)
	}
	if m.Replace != nil {
		fmt.Printf(" (replace: %s)", m.Replace.Path)
	}
	fmt.Println()
}
