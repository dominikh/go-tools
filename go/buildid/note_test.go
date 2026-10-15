package buildid

import (
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestReadELFRejectsOverflowingNoteSize(t *testing.T) {
	data := make([]byte, 137)
	copy(data[:4], []byte{0x7f, 'E', 'L', 'F'})
	data[elf.EI_CLASS] = byte(elf.ELFCLASS64)
	data[elf.EI_DATA] = byte(elf.ELFDATA2LSB)
	data[elf.EI_VERSION] = byte(elf.EV_CURRENT)
	binary.LittleEndian.PutUint16(data[16:], uint16(elf.ET_EXEC))
	binary.LittleEndian.PutUint16(data[18:], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(data[20:], uint32(elf.EV_CURRENT))
	binary.LittleEndian.PutUint64(data[32:], 64) // program header offset
	binary.LittleEndian.PutUint16(data[52:], 64) // ELF header size
	binary.LittleEndian.PutUint16(data[54:], 56) // program header entry size
	binary.LittleEndian.PutUint16(data[56:], 1)  // program header count
	binary.LittleEndian.PutUint32(data[64:], uint32(elf.PT_NOTE))
	binary.LittleEndian.PutUint64(data[72:], 0)
	binary.LittleEndian.PutUint64(data[80:], 120) // note offset
	binary.LittleEndian.PutUint64(data[96:], 16)  // file size
	binary.LittleEndian.PutUint64(data[104:], 16) // memory size
	binary.LittleEndian.PutUint64(data[112:], 4)  // alignment
	binary.LittleEndian.PutUint32(data[120:], 4)  // name size
	binary.LittleEndian.PutUint32(data[124:], 0xfffffff0)
	binary.LittleEndian.PutUint32(data[128:], 4) // Go build ID note type
	copy(data[132:136], elfGoNote)

	filename := filepath.Join(t.TempDir(), "malformed.elf")
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := readELF(filename, f, data); err != nil {
		t.Fatalf("readELF returned error for malformed note: %v", err)
	}
}
