package pkg

import (
	"encoding/binary"
	"io"
	"log"
)

func fn() {
	var x bool
	log.Println(binary.Write(io.Discard, binary.LittleEndian, x)) //@ diag(`cannot be used with binary.Write`)
}
