// Test of field usage detection

package pkg

type t1 struct { //@ used("t1", true)
	f11 int //@ used("f11", true)
	f12 int //@ used("f12", true)
}
type t2 struct { //@ used("t2", true)
	f21 int //@ used("f21", true)
	f22 int //@ used("f22", true)
}
type t3 struct { //@ used("t3", true)
	f31 t4 //@ used("f31", true)
}
type t4 struct { //@ used("t4", true)
	f41 int //@ used("f41", true)
}
type t5 struct { //@ used("t5", true)
	f51 int //@ used("f51", true)
}
type t6 struct { //@ used("t6", true)
	f61 int //@ used("f61", true)
}
type t7 struct { //@ used("t7", true)
	f71 int //@ used("f71", true)
}
type m1 map[string]t7 //@ used("m1", true)
type t8 struct {      //@ used("t8", true)
	f81 int //@ used("f81", true)
}
type t9 struct { //@ used("t9", true)
	f91 int //@ used("f91", true)
}
type t10 struct { //@ used("t10", true)
	f101 int //@ used("f101", true)
}
type t11 struct { //@ used("t11", true)
	f111 int //@ used("f111", true)
}
type s1 []t11     //@ used("s1", true)
type t12 struct { //@ used("t12", true)
	f121 int //@ used("f121", true)
}
type s2 []t12     //@ used("s2", true)
type t13 struct { //@ used("t13", true)
	f131 int //@ used("f131", true)
}
type t14 struct { //@ used("t14", true)
	f141 int //@ used("f141", true)
}
type a1 [1]t14    //@ used("a1", true)
type t15 struct { //@ used("t15", true)
	f151 int //@ used("f151", true)
}
type a2 [1]t15    //@ used("a2", true)
type t16 struct { //@ used("t16", true)
	f161 int //@ used("f161", true)
}
type t17 struct { //@ used("t17", false)
	f171 int //@ quiet("f171")
	f172 int //@ quiet("f172")
}
type t18 struct { //@ used("t18", true)
	f181 int //@ used("f181", true)
	f182 int //@ used("f182", false)
	f183 int //@ used("f183", false)
}

type t19 struct { //@ used("t19", true)
	f191 int //@ used("f191", true)
}
type m2 map[string]t19 //@ used("m2", true)

type t20 struct { //@ used("t20", true)
	f201 int //@ used("f201", true)
}
type m3 map[string]t20 //@ used("m3", true)

type t21 struct { //@ used("t21", true)
	f211 int //@ used("f211", false)
	f212 int //@ used("f212", true)
}
type t22 struct { //@ used("t22", false)
	f221 int //@ quiet("f221")
	f222 int //@ quiet("f222")
}

func foo() { //@ used("foo", true)
	_ = t10{1}
	_ = t21{f212: 1}
	_ = []t1{{1, 2}}
	_ = t2{1, 2}
	_ = []struct {
		a int //@ used("a", true)
	}{{1}}

	// XXX
	// _ = []struct{ foo struct{ bar int } }{{struct{ bar int }{1}}}

	_ = []t1{t1{1, 2}}
	_ = []t3{{t4{1}}}
	_ = map[string]t5{"a": {1}}
	_ = map[t6]string{{1}: "a"}
	_ = m1{"a": {1}}
	_ = map[t8]t8{{}: {1}}
	_ = map[t9]t9{{1}: {}}
	_ = s1{{1}}
	_ = s2{2: {1}}
	_ = [...]t13{{1}}
	_ = a1{{1}}
	_ = a2{0: {1}}
	_ = map[[1]t16]int{{{1}}: 1}
	y := struct { //@ used("y", true)
		x int //@ used("x", true)
	}{}
	_ = y
	_ = t18{f181: 1}
	_ = []m2{{"a": {1}}}
	_ = [][]m3{{{"a": {1}}}}
}

func init() { foo() } //@ used("init", true)

func superUnused() { //@ used("superUnused", false)
	var _ struct { //@ quiet("_")
		x int //@ quiet("x")
	}
}
