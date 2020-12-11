// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snowman

import (
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow/consensus/snowman"
	"github.com/liraxapp/avalanchego/snow/engine/common"
	"github.com/liraxapp/avalanchego/utils/wrappers"
)

// convincer sends chits to [vdr] once all its dependencies are met
type convincer struct {
	consensus snowman.Consensus
	sender    common.Sender
	vdr       ids.ShortID
	requestID uint32
	abandoned bool
	deps      ids.Set
	errs      *wrappers.Errs
}

func (c *convincer) Dependencies() ids.Set { return c.deps }

// Mark that a dependency has been met
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

	pref := []ids.ID{c.consensus.Preference()}
	c.sender.Chits(c.vdr, c.requestID, pref)
}
