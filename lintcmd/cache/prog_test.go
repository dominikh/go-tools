package cache

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"honnef.co/go/tools/lintcmd/cache/cacheprog"
)

var errTestWriter = errors.New("test writer failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errTestWriter }

func TestWriteToChildReturnsBase64CloseError(t *testing.T) {
	const groups = 1000
	bodySize := groups*3 + 1 // The base64 encoder buffers a final padded group.
	body := strings.Repeat("x", bodySize)
	req := &cacheprog.Request{
		ID:       1,
		Command:  cacheprog.CmdPut,
		ActionID: []byte{1},
		OutputID: []byte{2},
		BodySize: int64(bodySize),
	}
	var header bytes.Buffer
	if err := json.NewEncoder(&header).Encode(req); err != nil {
		t.Fatal(err)
	}
	// Leave three bytes available after the request header and the unpadded
	// base64 body, so the final padded group makes the buffer flush and fail
	// from Encoder.Close.
	bufferSize := header.Len() + 2 + groups*4 + 3
	pc := &ProgCache{
		inFlight: map[int64]chan<- *cacheprog.Response{},
		bw:       bufio.NewWriterSize(failingWriter{}, bufferSize),
	}
	pc.jenc = json.NewEncoder(pc.bw)
	got := pc.writeToChild(&cacheprog.Request{
		Command:  cacheprog.CmdPut,
		ActionID: []byte{1},
		OutputID: []byte{2},
		Body:     strings.NewReader(body),
		BodySize: int64(bodySize),
	}, make(chan *cacheprog.Response, 1))
	if !errors.Is(got, errTestWriter) {
		t.Fatalf("writeToChild error = %v, want %v", got, errTestWriter)
	}
}
