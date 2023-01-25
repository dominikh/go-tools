package pkg

import (
	"fmt"
	"log"
	"os"
)

func fn(s string) {
	fn2 := func() string { return "" }
	fmt.Printf(fn2())      //@ diag(`should use print-style function`)
	_ = fmt.Sprintf(fn2()) //@ diag(`should use print-style function`)
	log.Printf(fn2())      //@ diag(`should use print-style function`)
	fmt.Printf(s)          //@ diag(`should use print-style function`)
	fmt.Printf(s, "")
	fmt.Fprintf(os.Stdout, s) //@ diag(`should use print-style function`)
	fmt.Fprintf(os.Stdout, s, "")

	fmt.Printf(fn2(), "")
	fmt.Printf("")
	fmt.Printf("%s", "")
	fmt.Printf(fn3())
}

func fn3() (string, int) { return "", 0 }
