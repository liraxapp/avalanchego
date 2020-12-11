// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package secp256k1fx

import (
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow"
)

// ID that this Fx uses when labeled
var (
	ID = ids.ID{'s', 'e', 'c', 'p', '2', '5', '6', 'k', '1', 'f', 'x'}
)

// Factory ...
type Factory struct{}

// New ...
func (f *Factory) New(*snow.Context) (interface{}, error) { return &Fx{}, nil }
