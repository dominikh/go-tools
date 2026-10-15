package pkg

import bytesalias "bytes"

func fn() {
	_ = bytesalias.Compare(nil, nil) == 0 //@ diag(` bytes.Equal`)
	_ = bytesalias.Compare(nil, nil) != 0 //@ diag(`!bytes.Equal`)
	_ = bytesalias.Compare(nil, nil) > 0
	_ = bytesalias.Compare(nil, nil) < 0
}
