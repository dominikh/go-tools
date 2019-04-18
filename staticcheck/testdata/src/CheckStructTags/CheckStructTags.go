package pkg

type T1 struct {
	B int        `foo:"" foo:""` // MATCH "duplicate struct tag"
	C int        `foo:"" bar:""`
	D int        `json:"-"`
	E int        `json:"\\"`                   // MATCH "invalid JSON field name"
	F int        `json:",omitempty,omitempty"` // MATCH "duplicate JSON option "omitempty""
	G int        `json:",omitempty,string"`
	H int        `json:",string,omitempty,string"` // MATCH "duplicate JSON option "string""
	I int        `json:",unknown"`                 // MATCH "unknown JSON option "unknown""
	J int        `json:",string"`
	K *int       `json:",string"`
	L **int      `json:",string"` // MATCH "the JSON string option"
	M complex128 `json:",string"` // MATCH "the JSON string option"
	N int        `json:"some-name"`
	O int        `json:"some-name,inline"`
}

type T2 struct {
	A int `xml:",attr"`
	B int `xml:",chardata"`
	C int `xml:",cdata"`
	D int `xml:",innerxml"`
	E int `xml:",comment"`
	F int `xml:",omitempty"`
	G int `xml:",any"`
	H int `xml:",unknown"` // MATCH "unknown XML option"
	I int `xml:",any,any"` // MATCH "duplicate XML option"
	J int `xml:"a>b>c,"`
	K int `xml:",attr,cdata"` // MATCH "mutually exclusive"
}

type T3 struct {
	A int `json:",omitempty" xml:",attr"`
	B int `json:",unknown" xml:",attr"` // MATCH "unknown JSON option"
}
