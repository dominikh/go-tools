package pkg

import (
	"fmt"
	"log"
	"testing"
)

func fn() {
	fmt.Print(fmt.Sprintf("%d", 1))         // want `should use fmt\.Printf`
	fmt.Println(fmt.Sprintf("%d", 1))       // want `don't forget the newline`
	fmt.Fprint(nil, fmt.Sprintf("%d", 1))   // want `should use fmt\.Fprintf`
	fmt.Fprintln(nil, fmt.Sprintf("%d", 1)) // want `don't forget the newline`
	fmt.Sprint(fmt.Sprintf("%d", 1))        // want `should use fmt\.Sprintf`
	fmt.Sprintln(fmt.Sprintf("%d", 1))      // want `don't forget the newline`

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
	t.Error(fmt.Sprintf(""))  // want `use t\.Errorf\(\.\.\.\) instead of t\.Error\(fmt\.Sprintf\(\.\.\.\)\)`
	b.Error(fmt.Sprintf(""))  // want `use b\.Errorf`
	tb.Error(fmt.Sprintf("")) // want `use tb\.Errorf`
	t.Fatal(fmt.Sprintf(""))  // want `use t\.Fatalf`
	b.Fatal(fmt.Sprintf(""))  // want `use b\.Fatalf`
	tb.Fatal(fmt.Sprintf("")) // want `use tb\.Fatalf`
	t.Log(fmt.Sprintf(""))    // want `use t\.Logf`
	b.Log(fmt.Sprintf(""))    // want `use b\.Logf`
	tb.Log(fmt.Sprintf(""))   // want `use tb\.Logf`
	t.Skip(fmt.Sprintf(""))   // want `use t\.Skipf`
	b.Skip(fmt.Sprintf(""))   // want `use b\.Skipf`
	tb.Skip(fmt.Sprintf(""))  // want `use tb\.Skipf`

	var e1 Embedding1
	var e2 Embedding2
	var e3 Embedding3
	var e4 Embedding4
	// Error and Errorf are both of *testing.common -> flag
	e1.Error(fmt.Sprintf("")) // want `use e1\.Errorf`
	// Fatal and Fatalf are both of *testing.common -> flag
	e1.Fatal(fmt.Sprintf("")) // want `use e1\.Fatalf`
	// Error is of *testing.common, but Errorf is Embedding2.Errorf -> don't flag
	e2.Error(fmt.Sprintf(""))
	// Fatal and Fatalf are both of *testing.common -> flag
	e2.Fatal(fmt.Sprintf("")) // want `use e2\.Fatalf`
	// Error is Embedding3.Error and Errorf is of *testing.common -> don't flag
	e3.Error(fmt.Sprintf(""))
	// Fatal and Fatalf are both of *testing.common -> flag
	e3.Fatal(fmt.Sprintf("")) // want `use e3\.Fatalf`
	// Error and Errorf are both of testing.TB -> flag
	e4.Error(fmt.Sprintf("")) // want `use e4\.Errorf`
	// Fatal and Fatalf are both of testing.TB -> flag
	e4.Fatal(fmt.Sprintf("")) // want `use e4\.Fatalf`

	// Basic cases
	log.Fatal(fmt.Sprintf(""))   // want `use log.Fatalf`
	log.Fatalln(fmt.Sprintf("")) // want `use log.Fatalf`
	log.Panic(fmt.Sprintf(""))   // want `use log.Panicf`
	log.Panicln(fmt.Sprintf("")) // want `use log.Panicf`
	log.Print(fmt.Sprintf(""))   // want `use log.Printf`
	log.Println(fmt.Sprintf("")) // want `use log.Printf`

	var l *log.Logger
	l.Fatal(fmt.Sprintf(""))   // want `use l.Fatalf\(\.\.\.\) instead of l\.Fatal\(fmt\.Sprintf\(\.\.\.\)`
	l.Fatalln(fmt.Sprintf("")) // want `use l.Fatalf`
	l.Panic(fmt.Sprintf(""))   // want `use l.Panicf`
	l.Panicln(fmt.Sprintf("")) // want `use l.Panicf`
	l.Print(fmt.Sprintf(""))   // want `use l.Printf`
	l.Println(fmt.Sprintf("")) // want `use l.Printf`

	// log.Logger and testing.T share a code path, no need to check embedding again
}
