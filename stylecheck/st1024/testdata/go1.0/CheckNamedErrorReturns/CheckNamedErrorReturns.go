package pkg

// no names
func fnA0()                {}
func fnA1() error          { return nil }
func fnA2() (int, error)   { return 0, nil }
func fnA3() (error, int)   { return nil, 0 }
func fnA4() (error, error) { return nil, nil }

// names, but not 'err'
func fnB1() (a error)          { return nil }
func fnB2() (a int, b error)   { return 0, nil }
func fnB3() (a error, b int)   { return nil, 0 }
func fnB4() (a error, b error) { return nil, nil }

// names, and '_'
func fnC1() (_ error)          { return nil }
func fnC2() (a int, _ error)   { return 0, nil }
func fnC3() (_ error, b int)   { return nil, 0 }
func fnC4() (_ error, b error) { return nil, nil }
func fnC5() (a error, _ error) { return nil, nil }

// false positives: non-errors named 'err'
func fnE1() (err int)          { return 0 }
func fnE2() (err int, b error) { return 0, nil }
func fnE3() (a error, err int) { return nil, 0 }

// bad ones: errors named 'err'
func fnD1() (err error)          { return nil }      //@ diag(`named error return should not be named 'err'`)
func fnD2() (a int, err error)   { return 0, nil }   //@ diag(`named error return should not be named 'err'`)
func fnD3() (err error, b int)   { return nil, 0 }   //@ diag(`named error return should not be named 'err'`)
func fnD4() (err error, b error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)
func fnD5() (a error, err error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)

type structA struct{}

// no names
func (structA) fnA0()                {}
func (structA) fnA1() error          { return nil }
func (structA) fnA2() (int, error)   { return 0, nil }
func (structA) fnA3() (error, int)   { return nil, 0 }
func (structA) fnA4() (error, error) { return nil, nil }

// names, but not 'err'
func (structA) fnB1() (a error)          { return nil }
func (structA) fnB2() (a int, b error)   { return 0, nil }
func (structA) fnB3() (a error, b int)   { return nil, 0 }
func (structA) fnB4() (a error, b error) { return nil, nil }

// names, and '_'
func (structA) fnC1() (_ error)          { return nil }
func (structA) fnC2() (a int, _ error)   { return 0, nil }
func (structA) fnC3() (_ error, b int)   { return nil, 0 }
func (structA) fnC4() (_ error, b error) { return nil, nil }
func (structA) fnC5() (a error, _ error) { return nil, nil }

// false positives: non-errors named 'err'
func (structA) fnE1() (err int)          { return 0 }
func (structA) fnE2() (err int, b error) { return 0, nil }
func (structA) fnE3() (a error, err int) { return nil, 0 }

// bad ones: errors named 'err'
func (structA) fnD1() (err error)          { return nil }      //@ diag(`named error return should not be named 'err'`)
func (structA) fnD2() (a int, err error)   { return 0, nil }   //@ diag(`named error return should not be named 'err'`)
func (structA) fnD3() (err error, b int)   { return nil, 0 }   //@ diag(`named error return should not be named 'err'`)
func (structA) fnD4() (err error, b error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)
func (structA) fnD5() (a error, err error) { return nil, nil } //@ diag(`named error return should not be named 'err'`)
