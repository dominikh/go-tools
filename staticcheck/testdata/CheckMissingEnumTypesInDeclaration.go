package pkg

const c1 int = 0
const c2 = 0

const (
	c3 int = iota
	c4
	c5
)

const ( // MATCH "only the first constant has an explicit type"
	c6 int = 1
	c7     = 2
	c8     = 3
)

const (
	c9  int = 1
	c10     = 2
	c11     = 3
	c12 int = 4
)

const (
	c13     = 1
	c14 int = 2
	c15 int = 3
	c16 int = 4
)

const (
	c17     = 1
	c18 int = 2
	c19     = 3
	c20 int = 4
)
