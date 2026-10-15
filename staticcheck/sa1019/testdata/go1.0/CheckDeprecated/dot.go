package pkg

import . "example.com/CheckDeprecated.assist_external"

func dotImported() {
	DeprecatedFn()    //@ diag(`pkg.DeprecatedFn is deprecated: use another function`)
	_ = SD{D: "used"} //@ diag(`external don't use me`)
}
