package pkg

func fn() int { return 0 }

var x = fn
var _ = x()
