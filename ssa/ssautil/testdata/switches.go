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
	// switch t9 {
	// case 1:int: call print(1:int) t8
	// case 2:int: call print(23:int) t8
	// case 3:int: call print(23:int) t8
	// case 4:int: phi [3: t18, 7: t8] #<mem>
	// default: t9 == t10
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
	// switch t9 {
	// case 1:int: call print(1:int) t8
	// case 2:int: call print(23:int) t8
	// case 3:int: call print(23:int) t8
	// default: call four() t8
	// }

	// switch t9 {
	// case 5:int: call print(5:int) t25
	// case 6:int: call print(6:int) t25
	// default: phi [2: t15, 3: t17, 5: t21, 8: t30, 10: t34, 11: t25] #<mem>
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
	// switch t13 {
	// case 1:int: call print(12:int) t12
	// case 2:int: call print(12:int) t12
	// default: t13 < 5:int
	// }
	if x == 1 || 2 == x || x < 5 {
		print(12)
	}

	// switch t13 {
	// case 3:int: call print(34:int) t19
	// case 4:int: call print(34:int) t19
	// default: t13 == t14
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
	// switch t5 {
	// case 1:int: call print(1:int) t4
	// case 2:int: call print(2:int) t4
	// default: call print("else":string) t4
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
	// switch t5 {
	// case 1:int: phi [0: t4, 4: t15] #<mem>
	// case 2:int: call print(2:int) t4
	// default: call print("else":string) t4
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
	// switch t5 {
	// case 1:int: call print(1:int) t8
	// case 2:int: call print(2:int) t8
	// default: phi [0: t4, 5: t8] #<mem>
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
	// switch t5 {
	// case 1:int: call print(1:int) t8
	// case 2:int: call print(2:int) t8
	// default: phi [0: t4, 3: t8] #<mem>
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
	// switch t5 {
	// case 1:int: call print(1:int) t4
	// case 2:int: t5 == 1:int
	// default: call print("end":string) t4
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
	// switch t5 {
	// case 1:int: call print(1:int) t4
	// case 1:int: call print("1a":string) t4
	// case 2:int: call print(2:int) t4
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
	// switch t6 {
	// case 0:int: call print(1:int) t5
	// case 2:int: call print(2:int) t5
	// default: phi [1: t9, 3: t5, 4: t16] #<mem>
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
	// switch t8 {
	// case 0:int: extract t7 #3
	// case 1:int: call println(0:int) t9
	// case 2:int: call println(1:int) t9
	// default: call println("default":string) t9
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
	// switch t8 {
	// case 0:int: extract t7 #3
	// case 1:int: call println(0:int) t9
	// case 2:int: call println(1:int) t9
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
	// switch t3.(type) {
	// case t10 int: call println(t3) t2
	// case t16 bool: call println(t3) t2
	// case t20 string: call println(t20) t2
	// default: call println(t3) t2
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
	// switch t4.(type) {
	// case t6 string: call println(1:int) t3
	// case t13 interface{}: call println(t13) t3
	// case t19 int: call println(3:int) t3
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
	// switch t3.(type) {
	// case t5 int: call println(t5) t2
	// case t12 string: call println(t12) t2
	// default: call print("default":string) t2
	// }
	if i, ok := x.(int); ok {
		println(i)
	} else if s, ok := x.(string); ok {
		println(s)
	} else {
		print("default")
	}
}
