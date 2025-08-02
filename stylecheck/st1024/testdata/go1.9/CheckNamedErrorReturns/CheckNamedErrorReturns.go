package pkg

type Error = error

// no names
func fnA0()                {}
func fnA1() Error          { return nil }
func fnA2() (int, Error)   { return 0, nil }
func fnA3() (Error, int)   { return nil, 0 }
func fnA4() (Error, Error) { return nil, nil }

// names, but not 'err'
func fnB1() (a Error)          { return nil }
func fnB2() (a int, b Error)   { return 0, nil }
func fnB3() (a Error, b int)   { return nil, 0 }
func fnB4() (a Error, b Error) { return nil, nil }

// names, and '_'
func fnC1() (_ Error)          { return nil }
func fnC2() (a int, _ Error)   { return 0, nil }
func fnC3() (_ Error, b int)   { return nil, 0 }
func fnC4() (_ Error, b Error) { return nil, nil }
func fnC5() (a Error, _ Error) { return nil, nil }

// false positives: non-errors named 'err'
func fnE1() (err int)          { return 0 }
func fnE2() (err int, b Error) { return 0, nil }
func fnE3() (a Error, err int) { return nil, 0 }

// bad ones: errors named 'err'
func fnD1() (err Error)          { return nil }      //@ diag(`named error return should not be named 'err'`)
func fnD2() (a int, err Error)   { return 0, nil }   //@ diag(`named error return should not be named 'err'`)
func fnD3() (err Error, b int)   { return nil, 0 }   //@ diag(`named error return should not be named 'err'`)
func fnD4() (err Error, b Error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)
func fnD5() (a Error, err Error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)

type structA struct{}

// no names
func (structA) fnA0()                {}
func (structA) fnA1() Error          { return nil }
func (structA) fnA2() (int, Error)   { return 0, nil }
func (structA) fnA3() (Error, int)   { return nil, 0 }
func (structA) fnA4() (Error, Error) { return nil, nil }

// names, but not 'err'
func (structA) fnB1() (a Error)          { return nil }
func (structA) fnB2() (a int, b Error)   { return 0, nil }
func (structA) fnB3() (a Error, b int)   { return nil, 0 }
func (structA) fnB4() (a Error, b Error) { return nil, nil }

// names, and '_'
func (structA) fnC1() (_ Error)          { return nil }
func (structA) fnC2() (a int, _ Error)   { return 0, nil }
func (structA) fnC3() (_ Error, b int)   { return nil, 0 }
func (structA) fnC4() (_ Error, b Error) { return nil, nil }
func (structA) fnC5() (a Error, _ Error) { return nil, nil }

// false positives: non-errors named 'err'
func (structA) fnE1() (err int)          { return 0 }
func (structA) fnE2() (err int, b Error) { return 0, nil }
func (structA) fnE3() (a Error, err int) { return nil, 0 }

// bad ones: errors named 'err'
func (structA) fnD1() (err Error)          { return nil }      //@ diag(`named error return should not be named 'err'`)
func (structA) fnD2() (a int, err Error)   { return 0, nil }   //@ diag(`named error return should not be named 'err'`)
func (structA) fnD3() (err Error, b int)   { return nil, 0 }   //@ diag(`named error return should not be named 'err'`)
func (structA) fnD4() (err Error, b Error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)
func (structA) fnD5() (a Error, err Error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)
