package pkg

import (
	"flag"
	"os"
)

type T struct {
	Usage func()
}

func fn1() {
	f := flag.NewFlagSet("", 0)
	f.Usage = func() {
		println("")
	}
	f.Usage = func() { // MATCH /the function assigned to Usage shouldn't call os.Exit/
		println("")
		os.Exit(1)
	}
	f.Usage = func() { // MATCH /the function assigned to Usage shouldn't call os.Exit/
		println("")
		if true {
			os.Exit(1)
		}
	}

	f = &flag.FlagSet{}
	f.Usage = func() { // MATCH /the function assigned to Usage shouldn't call os.Exit/
		os.Exit(1)
	}

	f2 := flag.FlagSet{}
	f2.Usage = func() { // MATCH /the function assigned to Usage shouldn't call os.Exit/
		os.Exit(1)
	}

	flag.Usage = func() { // MATCH /the function assigned to Usage shouldn't call os.Exit/
		os.Exit(1)
	}

	t := &T{}
	t.Usage = func() {
		os.Exit(1)
	}
}

func fn2(arg func()) {
	flag.Usage = arg
}
