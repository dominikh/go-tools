package pkg

import "strconv"

func ifn() int     { return 0 }
func ifn32() int32 { return 0 }
func ifn64() int64 { return 0 }

func fn() {
	var i int
	var i32 int32
	var i64 int64
	const c = 0
	const c32 int32 = 0
	const c64 int64 = 0

	strconv.FormatInt(int64(i), 10) // MATCH /strconv.Itoa instead of strconv.FormatInt/
	strconv.FormatInt(int64(i), 2)
	strconv.FormatInt(int64(i32), 10)
	strconv.FormatInt(int64(i64), 10)
	strconv.FormatInt(i64, 10)
	strconv.FormatInt(123, 10)        // MATCH /strconv.Itoa instead of strconv.FormatInt/
	strconv.FormatInt(int64(123), 10) // MATCH /strconv.Itoa instead of strconv.FormatInt/
	strconv.FormatInt(int64(999999999999999999), 10)
	strconv.FormatInt(int64(int64(123)), 10)
	strconv.FormatInt(c, 10)          // MATCH /strconv.Itoa instead of strconv.FormatInt/
	strconv.FormatInt(int64(c), 10)   // MATCH /strconv.Itoa instead of strconv.FormatInt/
	strconv.FormatInt(int64(c32), 10) // MATCH /strconv.Itoa instead of strconv.FormatInt/
	strconv.FormatInt(c64, 10)
	strconv.FormatInt(int64(ifn()), 10) // MATCH /strconv.Itoa instead of strconv.FormatInt/
	strconv.FormatInt(int64(ifn32()), 10)
	strconv.FormatInt(int64(ifn64()), 10)
}
