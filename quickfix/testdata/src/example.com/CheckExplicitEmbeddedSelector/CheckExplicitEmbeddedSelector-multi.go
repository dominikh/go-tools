package pkg

type MultiLevel struct{ BasicOuter }

func fnMulti() {
	var multi MultiLevel
	_ = multi.BasicOuter.BasicInner.F1 //@ diag(`could remove embedded field "BasicOuter" from selector`), diag(`could remove embedded field "BasicInner" from selector`), diag(`could simplify selectors`)
	_ = multi.BasicOuter.F1            //@ diag(`could remove embedded field "BasicOuter" from selector`)
	_ = multi.BasicInner.F1            //@ diag(`could remove embedded field "BasicInner" from selector`)
	_ = multi.F1                       // minimal form
}
