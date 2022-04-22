// Test of field usage detection

package pkg

type t1 struct { //@ used(true)
	f11 int //@ used(true)
	f12 int //@ used(true)
}
type t2 struct { //@ used(true)
	f21 int //@ used(true)
	f22 int //@ used(true)
}
type t3 struct { //@ used(true)
	f31 t4 //@ used(true)
}
type t4 struct { //@ used(true)
	f41 int //@ used(true)
}
type t5 struct { //@ used(true)
	f51 int //@ used(true)
}
type t6 struct { //@ used(true)
	f61 int //@ used(true)
}
type t7 struct { //@ used(true)
	f71 int //@ used(true)
}
type m1 map[string]t7 //@ used(true)
type t8 struct {      //@ used(true)
	f81 int //@ used(true)
}
type t9 struct { //@ used(true)
	f91 int //@ used(true)
}
type t10 struct { //@ used(true)
	f101 int //@ used(true)
}
type t11 struct { //@ used(true)
	f111 int //@ used(true)
}
type s1 []t11     //@ used(true)
type t12 struct { //@ used(true)
	f121 int //@ used(true)
}
type s2 []t12     //@ used(true)
type t13 struct { //@ used(true)
	f131 int //@ used(true)
}
type t14 struct { //@ used(true)
	f141 int //@ used(true)
}
type a1 [1]t14    //@ used(true)
type t15 struct { //@ used(true)
	f151 int //@ used(true)
}
type a2 [1]t15    //@ used(true)
type t16 struct { //@ used(true)
	f161 int //@ used(true)
}
type t17 struct { //@ used(false)
	f171 int
	f172 int
}
type t18 struct { //@ used(true)
	f181 int //@ used(true)
	f182 int //@ used(false)
	f183 int //@ used(false)
}

type t19 struct { //@ used(true)
	f191 int //@ used(true)
}
type m2 map[string]t19 //@ used(true)

type t20 struct { //@ used(true)
	f201 int //@ used(true)
}
type m3 map[string]t20 //@ used(true)

type t21 struct { //@ used(true)
	f211 int //@ used(false)
	f212 int //@ used(true)
}
type t22 struct { //@ used(false)
	f221 int
	f222 int
}

func foo() { //@ used(true)
	_ = t10{1}
	_ = t21{f212: 1}
	_ = []t1{{1, 2}}
	_ = t2{1, 2}
	_ = []struct {
		a int //@ used(true)
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
	y := struct {
		x int //@ used(true)
	}{}
	_ = y
	_ = t18{f181: 1}
	_ = []m2{{"a": {1}}}
	_ = [][]m3{{{"a": {1}}}}
}

func init() { foo() } //@ used(true)

func superUnused() { //@ used(false)
	var _ struct {
		x int
	}
}
