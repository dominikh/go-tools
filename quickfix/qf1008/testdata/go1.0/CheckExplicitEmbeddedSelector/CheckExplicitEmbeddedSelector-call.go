package pkg

type FunctionCallOuter struct{ FunctionCallInner }
type FunctionCallInner struct {
	F8 func() FunctionCallContinuedOuter
}
type FunctionCallContinuedOuter struct{ FunctionCallContinuedInner }
type FunctionCallContinuedInner struct{ F9 int }

func fnCall() {
	var call FunctionCallOuter
	_ = call.FunctionCallInner.F8().FunctionCallContinuedInner.F9 //@ diag(`could remove embedded field "FunctionCallInner" from selector`), diag(`could remove embedded field "FunctionCallContinuedInner" from selector`), diag(`could simplify selectors`)
	_ = call.F8().F9                                              // minimal form
}
