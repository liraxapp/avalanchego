// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snowman

import (
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow"
	"github.com/liraxapp/avalanchego/snow/consensus/snowball"
)

// Consensus represents a general snowman instance that can be used directly to
// process a series of dependent operations.
type Consensus interface {
	// Takes in the context, snowball parameters, and the last accepted block.
	Initialize(*snow.Context, snowball.Parameters, ids.ID) error

	// Returns the parameters that describe this snowman instance
	Parameters() snowball.Parameters

	// Adds a new decision. Assumes the dependency has already been added.
	// Returns if a critical error has occurred.
	Add(Block) error

	// Issued returns true if the block has been issued into consensus
	Issued(Block) bool

	// Returns the ID of the tail of the strongly preferred sequence of
	// decisions.
	Preference() ids.ID

	// RecordPoll collects the results of a network poll. Assumes all decisions
	// have been previously added. Returns if a critical error has occurred.
	RecordPoll(ids.Bag) error

	// Finalized returns true if all decisions that have been added have been
	// finalized. Note, it is possible that after returning finalized, a new
	// decision may be added such that this instance is no longer finalized.
	Finalized() bool
}
