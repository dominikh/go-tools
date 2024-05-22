package pkg

type any = interface{}

func test() {
	_ = any((*int)(nil)) == nil //@ diag(`never true`)
	_ = any((error)(nil)) == nil
}
