package pkg

func fn1() {
	const s1 = "foobar"
	_ = "a" == s1              // MATCH /Comparing strings of different sizes/
	_ = "a"[:] == s1           // MATCH /Comparing strings of different sizes/
	_ = s1 == "a"              // MATCH /Comparing strings of different sizes/
	_ = "a"[:] == s1[:2]       // MATCH /Comparing strings of different sizes/
	_ = "ab"[:] == s1[1:2]     // MATCH /Comparing strings of different sizes/
	_ = "ab"[:] == s1[0+1:2]   // MATCH /Comparing strings of different sizes/
	_ = "a"[:] == "abc"        // MATCH /Comparing strings of different sizes/
	_ = "a"[:] == "a"+"bc"     // MATCH /Comparing strings of different sizes/
	_ = "foobar"[:] == s1+"bc" // MATCH /Comparing strings of different sizes/
	_ = "a"[:] == "abc"[:]     // MATCH /Comparing strings of different sizes/
	_ = "a"[:] == "abc"[:2]    // MATCH /Comparing strings of different sizes/

	_ = "abcdef"[:] == s1
	_ = "ab"[:] == s1[:2]
	_ = "a"[:] == s1[1:2]
	_ = "a"[:] == s1[0+1:2]
	_ = "abc"[:] == "abc"
	_ = "abc"[:] == "a"+"bc"
	_ = s1[:] == "foo"+"bar"
	_ = "abc"[:] == "abc"[:] // MATCH /identical expressions on the left and right side/
	_ = "ab"[:] == "abc"[:2]
}

/*
$ go run staticcheck.go -- ../../testdata/string-length-compare.go | less
../../testdata/string-length-compare.go:5:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:6:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:7:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:8:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:9:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:10:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:11:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:12:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:13:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:14:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:15:6: Comparing strings of different sizes for equality will always return false
../../testdata/string-length-compare.go:24:6: identical expressions on the left and right side of the '==' operator
*/
