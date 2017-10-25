package pkg

type t1 int
type Recv int

func Fn6() t1         { return 0 }
func Fn7() *t1        { return nil }
func (Recv) Fn10() t1 { return 0 }
