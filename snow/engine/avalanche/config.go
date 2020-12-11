// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avalanche

import (
	"github.com/liraxapp/avalanchego/snow/consensus/avalanche"
	"github.com/liraxapp/avalanchego/snow/engine/avalanche/bootstrap"
)

// Config wraps all the parameters needed for an avalanche engine
type Config struct {
	bootstrap.Config

	Params    avalanche.Parameters
	Consensus avalanche.Consensus
}
