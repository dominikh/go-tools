// +build ignore

package main

// This file is the input to TestSwitches in switch_test.go.
// Each multiway conditional with constant or type cases (Switch)
// discovered by Switches is printed, and compared with the
// comments.
//
// The body of each case is printed as the value of its first
// instruction.

// -------- Value switches --------

func SimpleSwitch(x, y int) {
	// switch t8 {
	// case 1:int: call print(1:int)
	// case 2:int: call print(23:int)
	// case 3:int: call print(23:int)
	// case 4:int: call print(3:int)
	// default: t8 == t9
	// }
	switch x {
	case 1:
		print(1)
	case 2, 3:
		print(23)
		fallthrough
	case 4:
		print(3)
	default:
		print(4)
	case y:
		print(5)
	}
	print(6)
}

func four() int { return 4 }

// A non-constant case makes a switch "impure", but its pure
// cases form two separate switches.
func SwitchWithNonConstantCase(x int) {
	// switch t8 {
	// case 1:int: call print(1:int)
	// case 2:int: call print(23:int)
	// case 3:int: call print(23:int)
	// default: call four()
	// }

	// switch t8 {
	// case 5:int: call print(5:int)
	// case 6:int: call print(6:int)
	// default: call print("done":string)
	// }
	switch x {
	case 1:
		print(1)
	case 2, 3:
		print(23)
	case four():
		print(3)
	case 5:
		print(5)
	case 6:
		print(6)
	}
	print("done")
}

// Switches may be found even where the source
// program doesn't have a switch statement.

func ImplicitSwitches(x, y int) {
	// switch t12 {
	// case 1:int: call print(12:int)
	// case 2:int: call print(12:int)
	// default: t12 < 5:int
	// }
	if x == 1 || 2 == x || x < 5 {
		print(12)
	}

	// switch t12 {
	// case 3:int: call print(34:int)
	// case 4:int: call print(34:int)
	// default: t12 == t13
	// }
	if x == 3 || 4 == x || x == y {
		print(34)
	}

	// Not a switch: no consistent variable.
	if x == 5 || y == 6 {
		print(56)
	}

	// Not a switch: only one constant comparison.
	if x == 7 || x == y {
		print(78)
	}
}

func IfElseBasedSwitch(x int) {
	// switch t4 {
	// case 1:int: call print(1:int)
	// case 2:int: call print(2:int)
	// default: call print("else":string)
	// }
	if x == 1 {
		print(1)
	} else if x == 2 {
		print(2)
	} else {
		print("else")
	}
}

func GotoBasedSwitch(x int) {
	// switch t4 {
	// case 1:int: call print(1:int)
	// case 2:int: call print(2:int)
	// default: call print("else":string)
	// }
	if x == 1 {
		goto L1
	}
	if x == 2 {
		goto L2
	}
	print("else")
L1:
	print(1)
	goto end
L2:
	print(2)
end:
}

func SwitchInAForLoop(x int) {
	// switch t4 {
	// case 1:int: call print(1:int)
	// case 2:int: call print(2:int)
	// default: call print("head":string)
	// }
loop:
	for {
		print("head")
		switch x {
		case 1:
			print(1)
			break loop
		case 2:
			print(2)
			break loop
		}
	}
}

// This case is a switch in a for-loop, both constructed using goto.
// As before, the default case points back to the block containing the
// switch, but that's ok.
func SwitchInAForLoopUsingGoto(x int) {
	// switch t4 {
	// case 1:int: call print(1:int)
	// case 2:int: call print(2:int)
	// default: call print("head":string)
	// }
loop:
	print("head")
	if x == 1 {
		goto L1
	}
	if x == 2 {
		goto L2
	}
	goto loop
L1:
	print(1)
	goto end
L2:
	print(2)
end:
}

func UnstructuredSwitchInAForLoop(x int) {
	// switch t4 {
	// case 1:int: call print(1:int)
	// case 2:int: t4 == 1:int
	// default: call print("end":string)
	// }
	for {
		if x == 1 {
			print(1)
			return
		}
		if x == 2 {
			continue
		}
		break
	}
	print("end")
}

func CaseWithMultiplePreds(x int) {
	for {
		if x == 1 {
			print(1)
			return
		}
	loop:
		// This block has multiple predecessors,
		// so can't be treated as a switch case.
		if x == 2 {
			goto loop
		}
		break
	}
	print("end")
}

func DuplicateConstantsAreNotEliminated(x int) {
	// switch t4 {
	// case 1:int: call print(1:int)
	// case 1:int: call print("1a":string)
	// case 2:int: call print(2:int)
	// default: return
	// }
	if x == 1 {
		print(1)
	} else if x == 1 { // duplicate => unreachable
		print("1a")
	} else if x == 2 {
		print(2)
	}
}

// Interface values (created by comparisons) are not constants,
// so ConstSwitch.X is never of interface type.
func MakeInterfaceIsNotAConstant(x interface{}) {
	if x == "foo" {
		print("foo")
	} else if x == 1 {
		print(1)
	}
}

func ZeroInitializedVarsAreConstants(x int) {
	// switch t5 {
	// case 0:int: call print(1:int)
	// case 2:int: call print(2:int)
	// default: call print("end":string)
	// }
	var zero int // SSA construction replaces zero with 0
	if x == zero {
		print(1)
	} else if x == 2 {
		print(2)
	}
	print("end")
}

// -------- Select --------

// NB, potentially fragile reliance on register number.
func SelectDesugarsToSwitch(ch chan int) {
	// switch t7 {
	// case 0:int: extract t6 #2
	// case 1:int: call println(0:int)
	// case 2:int: call println(1:int)
	// default: call println("default":string)
	// }
	select {
	case x := <-ch:
		println(x)
	case <-ch:
		println(0)
	case ch <- 1:
		println(1)
	default:
		println("default")
	}
}

// NB, potentially fragile reliance on register number.
func NonblockingSelectDefaultCasePanics(ch chan int) {
	// switch t7 {
	// case 0:int: extract t6 #2
	// case 1:int: call println(0:int)
	// case 2:int: call println(1:int)
	// default: make interface{} <- string ("blocking select m...":string)
	// }
	select {
	case x := <-ch:
		println(x)
	case <-ch:
		println(0)
	case ch <- 1:
		println(1)
	}
}

// -------- Type switches --------

// NB, reliance on fragile register numbering.
func SimpleTypeSwitch(x interface{}) {
	// switch t2.(type) {
	// case t9 int: call println(t2)
	// case t15 bool: call println(t2)
	// case t19 string: call println(t19)
	// default: call println(t2)
	// }
	switch y := x.(type) {
	case nil:
		println(y)
	case int, bool:
		println(y)
	case string:
		println(y)
	default:
		println(y)
	}
}

// NB, potentially fragile reliance on register number.
func DuplicateTypesAreNotEliminated(x interface{}) {
	// switch t3.(type) {
	// case t5 string: call println(1:int)
	// case t12 interface{}: call println(t12)
	// case t18 int: call println(3:int)
	// default: return
	// }
	switch y := x.(type) {
	case string:
		println(1)
	case interface{}:
		println(y)
	case int:
		println(3) // unreachable!
	}
}

// NB, potentially fragile reliance on register number.
func AdHocTypeSwitch(x interface{}) {
	// switch t2.(type) {
	// case t4 int: call println(t4)
	// case t11 string: call println(t11)
	// default: call print("default":string)
	// }
	if i, ok := x.(int); ok {
		println(i)
	} else if s, ok := x.(string); ok {
		println(s)
	} else {
		print("default")
	}
}
