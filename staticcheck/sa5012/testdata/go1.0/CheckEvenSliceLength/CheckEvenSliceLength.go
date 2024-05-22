package pkg

import "strings"

func fnVariadic(s string, args ...interface{}) { //@ fact(args, "needs even elements")
	if len(args)%2 != 0 {
		panic("I'm one of those annoying logging APIs")
	}
}

func fnSlice(s string, args []interface{}) { //@ fact(args, "needs even elements")
	if len(args)%2 != 0 {
		panic("I'm one of those annoying logging APIs")
	}
}

func fnIndirect(s string, args ...interface{}) { //@ fact(args, "needs even elements")
	fnSlice(s, args)
}

func fn2(bleh []interface{}, arr1 [3]interface{}) { //@ fact(bleh, "needs even elements")
	fnVariadic("%s", 1, 2, 3) //@ diag(re`variadic argument "args".+ but has 3 elements`)
	args := []interface{}{1, 2, 3}
	fnVariadic("", args...)     //@ diag(re`variadic argument "args".+ but has 3 elements`)
	fnVariadic("", args[:1]...) //@ diag(re`variadic argument "args".+ but has 1 elements`)
	fnVariadic("", args[:2]...)
	fnVariadic("", args[0:1]...) //@ diag(re`variadic argument "args".+ but has 1 elements`)
	fnVariadic("", args[0:]...)  //@ diag(re`variadic argument "args".+ but has 3 elements`)
	fnVariadic("", args[:]...)   //@ diag(re`variadic argument "args".+ but has 3 elements`)
	fnVariadic("", bleh...)
	fnVariadic("", bleh[:1]...)  //@ diag(re`variadic argument "args".+ but has 1 elements`)
	fnVariadic("", bleh[0:1]...) //@ diag(re`variadic argument "args".+ but has 1 elements`)
	fnVariadic("", bleh[0:]...)
	fnVariadic("", bleh[:]...)
	fnVariadic("", bleh)                      //@ diag(re`variadic argument "args".+ but has 1 elements`)
	fnVariadic("", make([]interface{}, 3)...) //@ diag(re`variadic argument "args".+ but has 3 elements`)
	fnVariadic("", make([]interface{}, 4)...)
	var arr2 [3]interface{}
	fnVariadic("", arr1[:]...) //@ diag(re`variadic argument "args".+ but has 3 elements`)
	fnVariadic("", arr2[:]...) //@ diag(re`variadic argument "args".+ but has 3 elements`)

	fnSlice("", []interface{}{1, 2, 3}) //@ diag(re`argument "args".+ but has 3 elements`)
	fnSlice("", []interface{}{1, 2, 3, 4})

	fnIndirect("%s", 1, 2, 3) //@ diag(re`argument "args".+ but has 3 elements`)
	fnIndirect("%s", 1, 2)

	strings.NewReplacer("one") //@ diag(re`variadic argument "oldnew".+ but has 1 elements`)
	strings.NewReplacer("one", "two")
}

func fn3() {
	args := []interface{}{""}
	if true {
		fnSlice("", args) //@ diag(`but has 1 element`)
	}
}
