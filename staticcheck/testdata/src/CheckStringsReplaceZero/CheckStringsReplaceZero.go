package pkg

import "strings"

func fn() {
	_ = strings.Replace("", "", "", 0) // MATCH /calling strings.Replace with n == 0/
	_ = strings.Replace("", "", "", -1)
	_ = strings.Replace("", "", "", 1)
}
