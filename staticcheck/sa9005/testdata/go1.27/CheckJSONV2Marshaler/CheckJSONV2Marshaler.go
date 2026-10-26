package pkg

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
)

type T struct {
	x int64
}

var (
	_ jsonv2.MarshalerTo     = T{}
	_ jsonv2.UnmarshalerFrom = (*T)(nil)
)

func (t T) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, t.x)
}

func (t *T) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return jsonv2.UnmarshalDecode(dec, &t.x)
}

func f() {
	t := T{}
	data, _ := json.Marshal(t)
	_ = json.Unmarshal(data, &t)
	(*json.Encoder)(nil).Encode(t)
	(*json.Decoder)(nil).Decode(&t)
}
