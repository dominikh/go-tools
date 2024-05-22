package pkg

func tpfn[T ~string | []byte | int](b T) {
	var m map[string]int
	k := string(b) //@ diag(`would be more efficient`)
	_ = m[k]
}
