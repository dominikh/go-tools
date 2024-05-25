package deferr

func x() {
	var (
		t                t1
		tt               t2
		varReturnNothing = func() {}
		varReturnInt     = func() int { return 0 }
		varReturnFunc    = func() func() { return func() {} }
		varReturnFuncInt = func(int) func(int) int { return func(int) int { return 0 } }
		varReturnMulti   = func() (int, func()) { return 0, func() {} }

		namedReturnNothing = named(func() {})
		namedReturnFunc    = namedReturn(func() func() { return func() {} })
	)

	// Correct.
	defer returnNothing()
	defer varReturnNothing()
	defer namedReturnNothing()
	defer t.returnNothing()
	defer tt.returnNothing()
	defer tt.t.returnNothing()
	defer func() {}()
	defer close(make(chan int))

	defer returnInt()
	defer varReturnInt()
	defer t.returnInt()
	defer tt.returnInt()
	defer tt.t.returnInt()
	defer func() int { return 0 }()

	defer returnFunc()()
	defer varReturnFunc()()
	defer namedReturnFunc()()
	defer t.returnFunc()()
	defer tt.returnFunc()()
	defer tt.t.returnFunc()()
	defer func() func() { return func() {} }()()

	defer returnFuncInt(0)(0)
	defer varReturnFuncInt(0)(0)
	defer t.returnFuncInt(0)(0)
	defer tt.returnFuncInt(0)(0)
	defer tt.t.returnFuncInt(0)(0)
	defer func(int) func(int) int { return func(int) int { return 0 } }(0)(0)

	defer returnMulti()
	defer varReturnMulti()
	defer t.returnMulti()
	defer tt.returnMulti()
	defer tt.t.returnMulti()
	defer func() (int, func()) { return 0, func() {} }()

	// Wrong.
	defer returnFunc()                                                     //@ diag(`defered return function not called`)
	defer varReturnFunc()                                                  //@ diag(`defered return function not called`)
	defer namedReturnFunc()                                                //@ diag(`defered return function not called`)
	defer t.returnFunc()                                                   //@ diag(`defered return function not called`)
	defer tt.returnFunc()                                                  //@ diag(`defered return function not called`)
	defer tt.t.returnFunc()                                                //@ diag(`defered return function not called`)
	defer func() func() { return func() {} }()                             //@ diag(`defered return function not called`)
	defer returnFuncInt(0)                                                 //@ diag(`defered return function not called`)
	defer t.returnFuncInt(0)                                               //@ diag(`defered return function not called`)
	defer tt.returnFuncInt(0)                                              //@ diag(`defered return function not called`)
	defer tt.t.returnFuncInt(0)                                            //@ diag(`defered return function not called`)
	defer func(int) func(int) int { return func(int) int { return 0 } }(0) //@ diag(`defered return function not called`)

	// Function returns a function which returns another function. This is
	// getting silly and is not checked.
	defer silly1()()
	defer func() func() func() {
		return func() func() {
			return func() {}
		}
	}()()
}

func returnNothing()                  {}
func returnInt() int                  { return 0 }
func returnFunc() func()              { return func() {} }
func returnFuncInt(int) func(int) int { return func(int) int { return 0 } }
func returnMulti() (int, func())      { return 0, func() {} }

type (
	t1          struct{}
	t2          struct{ t t1 }
	named       func()
	namedReturn func() func()
)

func (t1) returnNothing()                  {}
func (t1) returnInt() int                  { return 0 }
func (t1) returnFunc() func()              { return func() {} }
func (t1) returnFuncInt(int) func(int) int { return func(int) int { return 0 } }
func (t1) returnMulti() (int, func())      { return 0, func() {} }

func (*t2) returnNothing()                  {}
func (*t2) returnInt() int                  { return 0 }
func (*t2) returnFunc() func()              { return func() {} }
func (*t2) returnFuncInt(int) func(int) int { return func(int) int { return 0 } }
func (*t2) returnMulti() (int, func())      { return 0, func() {} }

func silly1() func() func() {
	return func() func() {
		return func() {}
	}
}
