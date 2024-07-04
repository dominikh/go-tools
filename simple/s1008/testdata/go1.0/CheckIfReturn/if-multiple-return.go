package pkg

func multireturn_fn1(x string) (bool, error) {
	if len(x) > 0 { //@ diag(`should use 'return len(x) > 0, nil' instead of 'if len(x) > 0 { return true, nil }; return false, nil'`)
		return true, nil
	}
	return false, nil
}

func multireturn_fn2(x string) (bool, error) {
	if len(x) > 0 {
	}
	return false, nil
}

func multireturn_fn3(x string) (bool, error, error) {
	if len(x) > 0 { //@ diag(`should use 'return len(x) > 0, nil, nil' instead of 'if len(x) > 0 { return true, nil, nil }; return false, nil, nil'`)
		return true, nil, nil
	}
	return false, nil, nil
}

func multireturn_fn4(x string) (bool, int, error) {
	if len(x) > 0 { //@ diag(`should use 'return len(x) == 0, 0, nil' instead of 'if len(x) > 0 { return false, 0, nil }; return true, 0, nil'`)
		return false, 0, nil
	}
	return true, 0, nil
}

func multireturn_fn5(x string) (bool, int, error) {
	if len(x) > 0 { 
		return false, 20, nil
	}
	return true, 30, nil
}
