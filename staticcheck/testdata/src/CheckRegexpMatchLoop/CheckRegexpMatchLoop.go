package pkg

import "regexp"

func fn() {
	regexp.Match(".", nil)
	regexp.MatchString(".", "")
	regexp.MatchReader(".", nil)

	for {
		regexp.Match(".", nil)       //@ diag(`calling regexp.Match in a loop has poor performance`)
		regexp.MatchString(".", "")  //@ diag(`calling regexp.MatchString in a loop has poor performance`)
		regexp.MatchReader(".", nil) //@ diag(`calling regexp.MatchReader in a loop has poor performance`)
	}
}
