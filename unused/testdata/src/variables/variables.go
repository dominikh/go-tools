package pkg

var a byte
var b [16]byte

func Fn() {
	println(a)
	_ = b[:]
}
