package pkg

type T5 struct {
	A int   `choice:"foo" choice:"bar"`                 //@ diag(`duplicate struct tag`)
	B []int `optional-value:"foo" optional-value:"bar"` //@ diag(`duplicate struct tag`)
	C []int `default:"foo" default:"bar"`               //@ diag(`duplicate struct tag`)
}
