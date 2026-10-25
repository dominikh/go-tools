package pkg

func other() {
	_ = -0.0 //@ diag(`does not produce a negative zero`)
	_ = float32(-0) //@ diag(`does not produce a negative zero`)
}
