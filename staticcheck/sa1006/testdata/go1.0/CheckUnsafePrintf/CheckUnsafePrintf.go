package pkg

import (
	"fmt"
	"log"
	"os"
	"testing"
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

	l := log.New(os.Stdout, "", 0)
	l.Printf("xx: %q", "yy")
	l.Printf(s) //@ diag(`should use print-style function`)

	var t testing.T
	t.Logf(fn2()) //@ diag(`should use print-style function`)
	t.Errorf(s)   //@ diag(`should use print-style function`)
	t.Fatalf(s)   //@ diag(`should use print-style function`)
	t.Skipf(s)    //@ diag(`should use print-style function`)

	var b testing.B
	b.Logf(fn2()) //@ diag(`should use print-style function`)
	b.Errorf(s)   //@ diag(`should use print-style function`)
	b.Fatalf(s)   //@ diag(`should use print-style function`)
	b.Skipf(s)    //@ diag(`should use print-style function`)

	var tb testing.TB
	tb.Logf(fn2()) //@ diag(`should use print-style function`)
	tb.Errorf(s)   //@ diag(`should use print-style function`)
	tb.Fatalf(s)   //@ diag(`should use print-style function`)
	tb.Skipf(s)    //@ diag(`should use print-style function`)

	fmt.Errorf(s) //@ diag(`should use print-style function`)

	var nested struct {
		l log.Logger
	}
	nested.l.Printf(s) //@ diag(`should use print-style function`)
}

func fn3() (string, int) { return "", 0 }
