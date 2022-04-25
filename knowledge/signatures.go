package knowledge

import "go/types"

var Signatures = map[string]*types.Signature{
	"(io.Seeker).Seek": types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewParam(0, nil, "", types.Universe.Lookup("int64").Type()),
			types.NewParam(0, nil, "", types.Universe.Lookup("int").Type()),
		),
		types.NewTuple(
			types.NewParam(0, nil, "", types.Universe.Lookup("int64").Type()),
			types.NewParam(0, nil, "", types.Universe.Lookup("error").Type()),
		),
		false,
	),
}
