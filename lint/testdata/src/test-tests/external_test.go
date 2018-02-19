package pkg

import "test-tests"

func fn3() { // MATCH "test problem"
	pkg.Exported()
}
