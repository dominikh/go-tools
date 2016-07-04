package pkg

import "errors"

func fn1() error {
	var err error

	if err != nil { // MATCH /simplified/
		return err
	}
	return nil

	_ = nil

	if err != nil { // MATCH /simplified/
		return err
	}
	return err
}

func fn2() (int, string, error) {
	var x int
	var y string
	var z int
	var err error

	if err != nil { // MATCH /simplified/
		return x, y, err
	}
	return x, y, nil

	_ = nil

	if err != nil { // MATCH /simplified/
		return x, y, err
	}
	return x, y, err

	_ = nil

	if err != nil {
		return x, y, err
	}
	return z, y, err

	_ = nil

	if err != nil {
		return 0, "", err
	}
	return x, y, err

	_ = nil

	// TODO(dominikh): currently, only returning identifiers is
	// supported
	if err != nil {
		return 42, "foo", err
	}
	return 42, "foo", err
}

func fn3() error {
	var err error
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func fn4(i int, err error) error {
	if err != nil {
		return err
	} else if i == 1 {
		return errors.New("some non-nil error")
	}
	return nil
}

func fn4() interface{} {
	var i *int
	if i != nil {
		return i
	}
	return nil

	var v interface{}
	if v != nil { // MATCH /simplified/
		return v
	}
	return nil
}

func fn5() {
	func() error {
		var err error
		if err != nil { // MATCH /simplified/
			return err
		}
		return nil
	}
}
