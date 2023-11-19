package pkg

import "testing"

func TestMain(m *testing.M) { //@ diag(`should call os.Exit`)
	m.Run()
}
