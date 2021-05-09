package pkg

type BasicOuter struct{ BasicInner }
type BasicInner struct{ F1 int }

func fnBasic() {
	var basic BasicOuter
	_ = basic.BasicInner.F1 // want `could remove anonymous field "BasicInner" from selector`
	_ = basic.F1            // minimal form
}
