package pkg

func fn(s string) {
	_ = string([]byte(s)) // MATCH "should use s instead of string([]byte(s))"
	_ = "" + s // MATCH /should use s instead of "" \+ s/
	_ = s + "" // MATCH /should use s instead of s \+ ""/

	_ = s
	_ = s + "foo"
	_ = s == ""
	_ = s != ""
	_ = "" +
		"really long lines follow" +
		"that need pretty formatting"

	_ = string([]rune(s))
	{
		string := func(v interface{}) string {
			return "foo"
		}
		_ = string([]byte(s))
	}
	{
		type byte rune
		_ = string([]byte(s))
	}
	{
		type T []byte
		var x T
		_ = string([]byte(x))
	}
}
