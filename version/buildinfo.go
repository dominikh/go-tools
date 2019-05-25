// +build go1.12

package version

import (
	"fmt"
	"runtime/debug"
)

func printBuildInfo() {
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Println("Main module:")
		printModule(&info.Main)
		fmt.Println("Dependencies:")
		for _, dep := range info.Deps {
			printModule(dep)
		}
	} else {
		fmt.Println("Built without Go modules")
	}
}

func buildInfoVersion() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	if info.Main.Version == "(devel)" {
		return "", false
	}
	return info.Main.Version, true
}
