//go:build go1.18

package pkg

func tpfn1[T string](x T) {
	for _, c := range []rune(x) { // want `should range over string`
		println(c)
	}
}

func tpfn2[T1 string, T2 []rune](x T1) {
	for _, c := range T2(x) { // want `should range over string`
		println(c)
	}
}
