package pkg

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

type countReadSeeker struct { //@ used_test(true)
	io.ReadSeeker       //@ used_test(true)
	N             int64 //@ used_test(true)
}

func (rs *countReadSeeker) Read(buf []byte) (int, error) { //@ used_test(true)
	n, err := rs.ReadSeeker.Read(buf)
	rs.N += int64(n)
	return n, err
}

func TestFoo(t *testing.T) { //@ used_test(true)
	r := bytes.NewReader([]byte("Hello, world!"))
	cr := &countReadSeeker{ReadSeeker: r}
	ioutil.ReadAll(cr)
	if cr.N != 13 {
		t.Errorf("got %d, want 13", cr.N)
	}
}

var sink int //@ used_test(true)

func BenchmarkFoo(b *testing.B) { //@ used_test(true)
	for i := 0; i < b.N; i++ {
		sink = fn()
	}
}

func fn() int { return 0 } //@ used_test(true)
