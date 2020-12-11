// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chains

import (
	"github.com/liraxapp/avalanchego/snow"
)

// Registrant can register the existence of a chain
type Registrant interface {
	RegisterChain(name string, ctx *snow.Context, vm interface{})
}
