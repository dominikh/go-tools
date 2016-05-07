package pkg

import "net/url"

func fn() {
	url.Parse("foobar")
	url.Parse(":") // MATCH /invalid argument to url.Parse/
	url.Parse("https://golang.org")
}
