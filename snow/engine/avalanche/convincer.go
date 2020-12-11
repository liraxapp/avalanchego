// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avalanche

import (
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow/consensus/avalanche"
	"github.com/liraxapp/avalanchego/snow/engine/common"
	"github.com/liraxapp/avalanchego/utils/wrappers"
)

// convincer sends chits to [vdr] after its dependencies are met.
type convincer struct {
	consensus avalanche.Consensus
	sender    common.Sender
	vdr       ids.ShortID
	requestID uint32
	abandoned bool
	deps      ids.Set
	errs      *wrappers.Errs
}

func (c *convincer) Dependencies() ids.Set { return c.deps }

// Mark that a dependency has been met.
func (c *convincer) Fulfill(id ids.ID) {
	c.deps.Remove(id)
	c.Update()
}

// Abandon this attempt to send chits.
func (c *convincer) Abandon(ids.ID) { c.abandoned = true }

func (c *convincer) Update() {
	if c.abandoned || c.deps.Len() != 0 || c.errs.Errored() {
		return
	}

	c.sender.Chits(c.vdr, c.requestID, c.consensus.Preferences().List())
}
