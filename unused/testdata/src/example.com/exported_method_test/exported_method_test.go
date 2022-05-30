package pkg

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

type countReadSeeker struct { //@ used_test("countReadSeeker", true)
	io.ReadSeeker       //@ used_test("ReadSeeker", true)
	N             int64 //@ used_test("N", true)
}

func (rs *countReadSeeker) Read(buf []byte) (int, error) { //@ used_test("Read", true), used_test("rs", true), used_test("buf", true)
	n, err := rs.ReadSeeker.Read(buf) //@ used_test("n", true), used_test("err", true)
	rs.N += int64(n)
	return n, err
}

func TestFoo(t *testing.T) { //@ used_test("TestFoo", true), used_test("t", true)
	r := bytes.NewReader([]byte("Hello, world!")) //@ used_test("r", true)
	cr := &countReadSeeker{ReadSeeker: r}         //@ used_test("cr", true)
	ioutil.ReadAll(cr)
	if cr.N != 13 {
		t.Errorf("got %d, want 13", cr.N)
	}
}

var sink int //@ used_test("sink", true)

func BenchmarkFoo(b *testing.B) { //@ used_test("BenchmarkFoo", true), used_test("b", true)
	for i := 0; i < b.N; i++ { //@ used_test("i", true)
		sink = fn()
	}
}

func fn() int { return 0 } //@ used_test("fn", true)
