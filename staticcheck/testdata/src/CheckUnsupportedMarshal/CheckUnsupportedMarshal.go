package pkg

import (
	"encoding/json"
	"encoding/xml"
)

type T1 struct {
	A int
	B func() `json:"-" xml:"-"`
	c chan int
}

type T2 struct {
	T1
}

type T3 struct {
	C chan int
}

type T4 struct {
	C C
}

type T5 struct {
	B func() `xml:"-"`
}

type T6 struct {
	B func() `json:"-"`
}

type C chan int

func (C) MarshalText() ([]byte, error) { return nil, nil }

func fn() {
	var t1 T1
	var t2 T2
	var t3 T3
	var t4 T4
	var t5 T5
	var t6 T6
	json.Marshal(t1)
	json.Marshal(t2)
	json.Marshal(t3) // MATCH "trying to marshal chan or func value"
	json.Marshal(t4)
	json.Marshal(t5) // MATCH "trying to marshal chan or func value"
	json.Marshal(t6)
	(*json.Encoder)(nil).Encode(t1)
	(*json.Encoder)(nil).Encode(t2)
	(*json.Encoder)(nil).Encode(t3) // MATCH "trying to marshal chan or func value"
	(*json.Encoder)(nil).Encode(t4)
	(*json.Encoder)(nil).Encode(t5) // MATCH "trying to marshal chan or func value"
	(*json.Encoder)(nil).Encode(t6)

	xml.Marshal(t1)
	xml.Marshal(t2)
	xml.Marshal(t3) // MATCH "trying to marshal chan or func value"
	xml.Marshal(t4)
	xml.Marshal(t5)
	xml.Marshal(t6) // MATCH "trying to marshal chan or func value"
	(*xml.Encoder)(nil).Encode(t1)
	(*xml.Encoder)(nil).Encode(t2)
	(*xml.Encoder)(nil).Encode(t3) // MATCH "trying to marshal chan or func value"
	(*xml.Encoder)(nil).Encode(t4)
	(*xml.Encoder)(nil).Encode(t5)
	(*xml.Encoder)(nil).Encode(t6) // MATCH "trying to marshal chan or func value"
}
