package pkg

import (
	"encoding/json"
	"encoding/xml"
	"time"
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
	Ch chan int
}

type T4 struct {
	C ValueMarshaler
}

type T5 struct {
	B func() `xml:"-"`
}

type T6 struct {
	B func() `json:"-"`
}

type T7 struct {
	A int
	B int
	T3
}

type T8 struct {
	C int
	*T7
}

type T9 struct {
	F PointerMarshaler
}

type T10 struct {
	F *struct {
		PointerMarshaler
	}
}

type Recursive struct {
	Field *Recursive
}

type ValueMarshaler chan int

func (ValueMarshaler) MarshalText() ([]byte, error) { return nil, nil }

type PointerMarshaler chan int

func (*PointerMarshaler) MarshalText() ([]byte, error) { return nil, nil }

func fn() {
	var t1 T1
	var t2 T2
	var t3 T3
	var t4 T4
	var t5 T5
	var t6 T6
	var t8 T8
	var t9 T9
	var t10 T10
	var t11 Recursive
	json.Marshal(t1)
	json.Marshal(t2)
	json.Marshal(t3) // want `unsupported type chan int, via x\.Ch`
	json.Marshal(t4)
	json.Marshal(t5) // want `unsupported type func\(\), via x\.B`
	json.Marshal(t6)
	(*json.Encoder)(nil).Encode(t1)
	(*json.Encoder)(nil).Encode(t2)
	(*json.Encoder)(nil).Encode(t3) // want `unsupported type chan int, via x\.Ch`
	(*json.Encoder)(nil).Encode(t4)
	(*json.Encoder)(nil).Encode(t5) // want `unsupported type func\(\), via x\.B`
	(*json.Encoder)(nil).Encode(t6)

	xml.Marshal(t1)
	xml.Marshal(t2)
	xml.Marshal(t3) // want `trying to marshal chan or func value, field CheckUnsupportedMarshal\.T3\.C`
	xml.Marshal(t4)
	xml.Marshal(t5)
	xml.Marshal(t6) // want `trying to marshal chan or func value, field CheckUnsupportedMarshal\.T6\.B`
	(*xml.Encoder)(nil).Encode(t1)
	(*xml.Encoder)(nil).Encode(t2)
	(*xml.Encoder)(nil).Encode(t3) // want `trying to marshal chan or func value, field CheckUnsupportedMarshal\.T3\.C`
	(*xml.Encoder)(nil).Encode(t4)
	(*xml.Encoder)(nil).Encode(t5)
	(*xml.Encoder)(nil).Encode(t6) // want `trying to marshal chan or func value, field CheckUnsupportedMarshal\.T6\.B`

	json.Marshal(t8)  // want `unsupported type chan int, via x\.T7\.T3\.Ch`
	json.Marshal(t9)  // want `unsupported type PointerMarshaler, via x\.F`
	json.Marshal(&t9) // this is fine, t9 is addressable, therefore T9.D is, too
	json.Marshal(t10) // this is fine, T10.F.D is addressable

	json.Marshal(t11)
	xml.Marshal(t11)
}

func addressability() {
	var a PointerMarshaler
	var b []PointerMarshaler
	var c struct {
		F PointerMarshaler
	}
	var d [4]PointerMarshaler
	json.Marshal(a) // want `unsupported type PointerMarshaler$`
	json.Marshal(&a)
	json.Marshal(b)
	json.Marshal(&b)
	json.Marshal(c) // want `unsupported type PointerMarshaler, via x\.F`
	json.Marshal(&c)
	json.Marshal(d) // want `unsupported type PointerMarshaler, via x\[0\]`
	json.Marshal(&d)

	var m1 map[string]PointerMarshaler
	json.Marshal(m1)                                // want `unsupported type PointerMarshaler, via x\[k\]`
	json.Marshal(&m1)                               // want `unsupported type PointerMarshaler, via x\[k\]`
	json.Marshal([]map[string]PointerMarshaler{m1}) // want `unsupported type PointerMarshaler, via x\[0\]\[k\]`

	var m2 map[string]*PointerMarshaler
	json.Marshal(m2)
	json.Marshal(&m2)
	json.Marshal([]map[string]*PointerMarshaler{m2})
}

func maps() {
	var good map[int]string
	var bad map[interface{}]string
	// the map key has to be statically known good; it must be a number or a string
	json.Marshal(good)
	json.Marshal(bad) // want `unsupported type map\[interface\{\}\]string$`

	var m1 map[string]PointerMarshaler
	json.Marshal(m1)                                // want `unsupported type PointerMarshaler, via x\[k\]`
	json.Marshal(&m1)                               // want `unsupported type PointerMarshaler, via x\[k\]`
	json.Marshal([]map[string]PointerMarshaler{m1}) // want `unsupported type PointerMarshaler, via x\[0\]\[k\]`

	var m2 map[string]*PointerMarshaler
	json.Marshal(m2)
	json.Marshal(&m2)
	json.Marshal([]map[string]*PointerMarshaler{m2})

	var m3 map[string]ValueMarshaler
	json.Marshal(m3)
	json.Marshal(&m3)
	json.Marshal([]map[string]ValueMarshaler{m3})

	var m4 map[string]*ValueMarshaler
	json.Marshal(m4)
	json.Marshal(&m4)
	json.Marshal([]map[string]*ValueMarshaler{m4})

	var m5 map[ValueMarshaler]string
	var m6 map[*ValueMarshaler]string
	var m7 map[PointerMarshaler]string
	var m8 map[*PointerMarshaler]string

	json.Marshal(m5)
	json.Marshal(m6)
	json.Marshal(m7) // want `unsupported type map\[PointerMarshaler\]string$`
	json.Marshal(m8)
}

func fieldPriority() {
	// In this example, the channel doesn't matter, because T1.F has higher priority than T1.T2.F
	type lT2 struct {
		F chan int
	}
	type lT1 struct {
		F int
		lT2
	}
	json.Marshal(lT1{})

	// In this example, it does matter
	type lT4 struct {
		C chan int
	}
	type lT3 struct {
		F int
		lT4
	}
	json.Marshal(lT3{}) // want `unsupported type chan int, via x\.lT4\.C`
}

func longPath() {
	var foo struct {
		Field struct {
			Field2 []struct {
				Map map[string]chan int
			}
		}
	}
	json.Marshal(foo) // want `unsupported type chan int, via x\.Field\.Field2\[0\].Map\[k\]`
}

func otherPackage() {
	var x time.Ticker
	json.Marshal(x) // want `unsupported type <-chan time\.Time, via x\.C`
}
