package pkg

import (
	"encoding/binary"
	"io"
)

func fn() {
	var x bool
	binary.Write(io.Discard, binary.LittleEndian, x)
}
