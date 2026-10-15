package pkg

import . "bytes"

func fn() {
	_ = Compare(nil, nil) == 0 //@ diag(`should use bytes.Equal`)
	_ = Compare(nil, nil) != 0 //@ diag(`should use !bytes.Equal`)
}
