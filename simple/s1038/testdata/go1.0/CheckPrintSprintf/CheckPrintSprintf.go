package pkg

import (
	"fmt"
	"log"
	"testing"
)

func fn() {
	fmt.Print(fmt.Sprintf("%d", 1))         //@ diag(`should use fmt.Printf`)
	fmt.Println(fmt.Sprintf("%d", 1))       //@ diag(`don't forget the newline`)
	fmt.Fprint(nil, fmt.Sprintf("%d", 1))   //@ diag(`should use fmt.Fprintf`)
	fmt.Fprintln(nil, fmt.Sprintf("%d", 1)) //@ diag(`don't forget the newline`)
	fmt.Sprint(fmt.Sprintf("%d", 1))        //@ diag(`should use fmt.Sprintf`)
	fmt.Sprintln(fmt.Sprintf("%d", 1))      //@ diag(`don't forget the newline`)

	arg := "%d"
	fmt.Println(fmt.Sprintf(arg, 1))
}

func Sprintf(string) string { return "" }

type Embedding1 struct {
	*testing.T
}

type Embedding2 struct {
	*testing.T
}

func (e Embedding2) Errorf() {}

type Embedding3 struct {
	*testing.T
}

func (e Embedding3) Error(string, ...interface{}) {}

type Embedding4 struct {
	testing.TB
}

func fn2() {
	var t *testing.T
	var b *testing.B
	var tb testing.TB

	// All of these are the basic cases that should be flagged
	t.Error(fmt.Sprintf(""))  //@ diag(`use t.Errorf(...) instead of t.Error(fmt.Sprintf(...))`)
	b.Error(fmt.Sprintf(""))  //@ diag(`use b.Errorf`)
	tb.Error(fmt.Sprintf("")) //@ diag(`use tb.Errorf`)
	t.Fatal(fmt.Sprintf(""))  //@ diag(`use t.Fatalf`)
	b.Fatal(fmt.Sprintf(""))  //@ diag(`use b.Fatalf`)
	tb.Fatal(fmt.Sprintf("")) //@ diag(`use tb.Fatalf`)
	t.Log(fmt.Sprintf(""))    //@ diag(`use t.Logf`)
	b.Log(fmt.Sprintf(""))    //@ diag(`use b.Logf`)
	tb.Log(fmt.Sprintf(""))   //@ diag(`use tb.Logf`)
	t.Skip(fmt.Sprintf(""))   //@ diag(`use t.Skipf`)
	b.Skip(fmt.Sprintf(""))   //@ diag(`use b.Skipf`)
	tb.Skip(fmt.Sprintf(""))  //@ diag(`use tb.Skipf`)

	var e1 Embedding1
	var e2 Embedding2
	var e3 Embedding3
	var e4 Embedding4
	// Error and Errorf are both of *testing.common -> flag
	e1.Error(fmt.Sprintf("")) //@ diag(`use e1.Errorf`)
	// Fatal and Fatalf are both of *testing.common -> flag
	e1.Fatal(fmt.Sprintf("")) //@ diag(`use e1.Fatalf`)
	// Error is of *testing.common, but Errorf is Embedding2.Errorf -> don't flag
	e2.Error(fmt.Sprintf(""))
	// Fatal and Fatalf are both of *testing.common -> flag
	e2.Fatal(fmt.Sprintf("")) //@ diag(`use e2.Fatalf`)
	// Error is Embedding3.Error and Errorf is of *testing.common -> don't flag
	e3.Error(fmt.Sprintf(""))
	// Fatal and Fatalf are both of *testing.common -> flag
	e3.Fatal(fmt.Sprintf("")) //@ diag(`use e3.Fatalf`)
	// Error and Errorf are both of testing.TB -> flag
	e4.Error(fmt.Sprintf("")) //@ diag(`use e4.Errorf`)
	// Fatal and Fatalf are both of testing.TB -> flag
	e4.Fatal(fmt.Sprintf("")) //@ diag(`use e4.Fatalf`)

	// Basic cases
	log.Fatal(fmt.Sprintf(""))   //@ diag(`use log.Fatalf`)
	log.Fatalln(fmt.Sprintf("")) //@ diag(`use log.Fatalf`)
	log.Panic(fmt.Sprintf(""))   //@ diag(`use log.Panicf`)
	log.Panicln(fmt.Sprintf("")) //@ diag(`use log.Panicf`)
	log.Print(fmt.Sprintf(""))   //@ diag(`use log.Printf`)
	log.Println(fmt.Sprintf("")) //@ diag(`use log.Printf`)

	var l *log.Logger
	l.Fatal(fmt.Sprintf(""))   //@ diag(`use l.Fatalf(...) instead of l.Fatal(fmt.Sprintf(...)`)
	l.Fatalln(fmt.Sprintf("")) //@ diag(`use l.Fatalf`)
	l.Panic(fmt.Sprintf(""))   //@ diag(`use l.Panicf`)
	l.Panicln(fmt.Sprintf("")) //@ diag(`use l.Panicf`)
	l.Print(fmt.Sprintf(""))   //@ diag(`use l.Printf`)
	l.Println(fmt.Sprintf("")) //@ diag(`use l.Printf`)

	// log.Logger and testing.T share a code path, no need to check embedding again
}
