package value

type Message interface {
	M()
}

type impl struct{}

func (*impl) M() {}

type Value struct{}

func (Value) Message() Message {
	println("not trivial")
	var p *impl
	return p
}
