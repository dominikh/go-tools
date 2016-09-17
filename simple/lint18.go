// +build go1.8

package simple

import "go/types"

// TODO(dh): use types.IdenticalIgnoreTags once CL 24190 has been merged
// var structsIdentical = types.IdenticalIgnoreTags
var structsIdentical = types.Identical
