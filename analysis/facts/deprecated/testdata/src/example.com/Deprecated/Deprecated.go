package pkg

// Deprecated: Don't use this.
func fn2() { // want fn2:`Deprecated: Don't use this\.`
}

// This is a function.
//
// Deprecated: Don't use this.
//
// Here is how you might use it instead.
func fn3() { // want fn3:`Deprecated: Don't use this\.`
}

// Handle cases like:
//
// Taken from "os" package:
//
// ```
// // Deprecated: Use io.SeekStart, io.SeekCurrent, and io.SeekEnd.
// const (
// 	SEEK_SET int = 0 // seek relative to the origin of the file
// 	SEEK_CUR int = 1 // seek relative to the current offset
// 	SEEK_END int = 2 // seek relative to the end
// )
// ```
//
// Here all three consts i.e., os.SEEK_SET, os.SEEK_CUR and os.SEEK_END are
// deprecated and not just os.SEEK_SET.

// Deprecated: Don't use this.
var (
	SEEK_A = 0 // want SEEK_A:`Deprecated: Don't use this\.`
	SEEK_B = 1 // want SEEK_B:`Deprecated: Don't use this\.`
	SEEK_C = 2 // want SEEK_C:`Deprecated: Don't use this\.`
)

// Deprecated: Don't use this.
type (
	pair struct{ x, y int }    // want pair:`Deprecated: Don't use this\.`
	cube struct{ x, y, z int } // want cube:`Deprecated: Don't use this\.`
)

// Deprecated: Don't use this.
var SEEK_D = 3 // want SEEK_D:`Deprecated: Don't use this\.`
var SEEK_E = 4
var SEEK_F = 5
