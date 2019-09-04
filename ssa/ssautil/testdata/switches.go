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
	// switch x {
	// case 1:int: call print(1:int) t0
	// case 2:int: call print(23:int) t0
	// case 3:int: call print(23:int) t0
	// case 4:int: phi [3: t5, 7: t0] #<mem>
	// default: x == y
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
	// switch x {
	// case 1:int: call print(1:int) t0
	// case 2:int: call print(23:int) t0
	// case 3:int: call print(23:int) t0
	// default: call four() t0
	// }

	// switch x {
	// case 5:int: call print(5:int) t9
	// case 6:int: call print(6:int) t9
	// default: phi [2: t4, 3: t5, 5: t7, 8: t13, 10: t15, 11: t9] #<mem>
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
	// switch x {
	// case 1:int: call print(12:int) t0
	// case 2:int: call print(12:int) t0
	// default: x < 5:int
	// }
	if x == 1 || 2 == x || x < 5 {
		print(12)
	}

	// switch x {
	// case 3:int: call print(34:int) t3
	// case 4:int: call print(34:int) t3
	// default: x == y
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
	// switch x {
	// case 1:int: call print(1:int) t0
	// case 2:int: call print(2:int) t0
	// default: call print("else":string) t0
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
	// switch x {
	// case 1:int: phi [0: t0, 4: t6] #<mem>
	// case 2:int: call print(2:int) t0
	// default: call print("else":string) t0
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
	// switch x {
	// case 1:int: call print(1:int) t2
	// case 2:int: call print(2:int) t2
	// default: phi [0: t0, 5: t2] #<mem>
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
	// switch x {
	// case 1:int: call print(1:int) t2
	// case 2:int: call print(2:int) t2
	// default: phi [0: t0, 3: t2] #<mem>
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
	// switch x {
	// case 1:int: call print(1:int) t0
	// case 2:int: x == 1:int
	// default: call print("end":string) t0
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
	// switch x {
	// case 1:int: call print(1:int) t0
	// case 1:int: call print("1a":string) t0
	// case 2:int: call print(2:int) t0
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
	// switch x {
	// case 0:int: call print(1:int) t0
	// case 2:int: call print(2:int) t0
	// default: phi [1: t2, 3: t0, 4: t6] #<mem>
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
	// switch t2 {
	// case 0:int: extract t1 #2
	// case 1:int: call println(0:int) t0
	// case 2:int: call println(1:int) t0
	// default: call println("default":string) t0
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
	// switch t2 {
	// case 0:int: extract t1 #2
	// case 1:int: call println(0:int) t0
	// case 2:int: call println(1:int) t0
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
	// switch x.(type) {
	// case t4 int: call println(x) t0
	// case t8 bool: call println(x) t0
	// case t11 string: call println(t11) t0
	// default: call println(x) t0
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
	// switch x.(type) {
	// case t2 string: call println(1:int) t0
	// case t6 interface{}: call println(t6) t0
	// case t10 int: call println(3:int) t0
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
	// switch x.(type) {
	// case t2 int: call println(t2) t0
	// case t6 string: call println(t6) t0
	// default: call print("default":string) t0
	// }
	if i, ok := x.(int); ok {
		println(i)
	} else if s, ok := x.(string); ok {
		println(s)
	} else {
		print("default")
	}
}
