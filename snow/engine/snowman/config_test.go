// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snowman

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/liraxapp/avalanchego/database/memdb"
	"github.com/liraxapp/avalanchego/snow/consensus/snowball"
	"github.com/liraxapp/avalanchego/snow/consensus/snowman"
	"github.com/liraxapp/avalanchego/snow/engine/common"
	"github.com/liraxapp/avalanchego/snow/engine/common/queue"
	"github.com/liraxapp/avalanchego/snow/engine/snowman/block"
	"github.com/liraxapp/avalanchego/snow/engine/snowman/bootstrap"
)

func DefaultConfig() Config {
	blocked, _ := queue.New(memdb.New())
	return Config{
		Config: bootstrap.Config{
			Config:  common.DefaultConfigTest(),
			Blocked: blocked,
			VM:      &block.TestVM{},
		},
		Params: snowball.Parameters{
			Metrics:           prometheus.NewRegistry(),
			K:                 1,
			Alpha:             1,
			BetaVirtuous:      1,
			BetaRogue:         2,
			ConcurrentRepolls: 1,
		},
		Consensus: &snowman.Topological{},
	}
}
