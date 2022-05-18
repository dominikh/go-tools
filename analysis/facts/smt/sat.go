package smt

/*
func (bl *builder) fullAdder(a, b, cIn Node) (s, cOut Node) {
	s = bl.Xor(bl.Xor(a, b), cIn)
	cOut = bl.Or(bl.And(bl.Xor(a, b), cIn), bl.And(a, b))
	return s, cOut
}

type BitVector struct {
	Base Raw
	Size int
}

func (bv BitVector) At(i int) Raw {
	return Raw(fmt.Sprintf("%s_%d", bv.Base, i))
}

func (bl *builder) adder(a, b BitVector) []Node {
	var c Node = Raw("false")
	out := make([]Node, a.Size)
	for i := 0; i < a.Size; i++ {
		out[i], c = bl.fullAdder(a.At(i), b.At(i), c)
	}
	return out
}
*/
