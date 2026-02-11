package pkg

import (
	"encoding/xml"

	_ "github.com/jessevdk/go-flags"
)

type T1 struct {
	B  int                    `foo:"" foo:""` //@ diag(`duplicate struct tag`)
	C  int                    `foo:"" bar:""`
	D  int                    `json:"-"`
	E  int                    `json:"\\"`                   //@ diag("malformed `json` tag")
	F  int                    `json:",omitempty,omitempty"` //@ diag("duplicate appearance of `omitempty` tag option")
	G  int                    `json:",omitempty,string"`
	H  int                    `json:",string,omitempty,string"` //@ diag("duplicate appearance of `string` tag option")
	I  int                    `json:",foreign"`                 //@ diag("invalid appearance of unknown `foreign` tag option")
	J  int                    `json:",string"`
	K  *int                   `json:",string"`
	L  **int                  `json:",string"` //@ diag("invalid appearance of `string` tag option")
	M  complex128             `json:",string"` //@ diag("invalid appearance of `string` tag option")
	N  int                    `json:"some-name"`
	O  int                    `json:"some-name,omitzero,omitempty,format:'something,with,commas'"`
	P  string                 `json:"-,omitempty"` //@ diag(`should encoding/json ignore this field or name it`)
	Q  string                 `json:"'-',omitempty"`
	R  map[string]interface{} `json:"unknown"`
	S  map[string]interface{} `json:"inline"`
	T2 `json:",omitzero"`
	T3 `json:"bar,omitzero"`
}

type T2 struct {
	A int `xml:",attr"`
	B int `xml:",chardata"`
	C int `xml:",cdata"`
	D int `xml:",innerxml"`
	E int `xml:",comment"`
	F int `xml:",omitempty"`
	G int `xml:",any"`
	H int `xml:",unknown"` //@ diag(`unknown option`)
	I int `xml:",any,any"` //@ diag(`duplicate option`)
	J int `xml:"a>b>c,"`
}

type T3 struct {
	A int `json:",omitempty" xml:",attr"`
	B int `json:",foreign" xml:",attr"` //@ diag("invalid appearance of unknown `foreign` tag option")
}

type T4 struct {
	A int   `choice:"foo" choice:"bar"`
	B []int `optional-value:"foo" optional-value:"bar"`
	C []int `default:"foo" default:"bar"`
	D int   `json:"foo" json:"bar"` //@ diag(`duplicate struct tag`)
}

func xmlTags() {
	type T1 struct {
		A       int      `xml:",attr,innerxml"` //@ diag(`invalid combination of options: ",attr,innerxml"`)
		XMLName xml.Name `xml:"ns "`            //@ diag(`namespace without name: "ns "`)
		B       int      `xml:"a>"`             //@ diag(`trailing '>'`)
		C       int      `xml:"a>b,attr"`       //@ diag(`a>b chain not valid with attr flag`)
	}
	type T6 struct {
		XMLName xml.Name `xml:"foo"`
	}
	type T5 struct {
		F T6 `xml:"f"` //@ diag(`name "f" conflicts with name "foo" in example.com/CheckStructTags.T6.XMLName`)
	}
}
