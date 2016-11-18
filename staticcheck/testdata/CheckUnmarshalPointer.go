package pkg

import "encoding/json"

func fn1() {
	var v map[string]interface{}
	var i interface{} = v
	p := &v
	json.Unmarshal([]byte(`{}`), v) // MATCH /Unmarshal expects to unmarshal into a pointer/
	json.Unmarshal([]byte(`{}`), &v)
	json.Unmarshal([]byte(`{}`), i) // we don't know what's in the interface
	json.Unmarshal([]byte(`{}`), p)

	json.NewDecoder(nil).Decode(v) // MATCH /Decode expects to unmarshal into a pointer/
}
