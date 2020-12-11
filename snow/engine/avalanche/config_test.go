// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avalanche

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/liraxapp/avalanchego/database/memdb"
	"github.com/liraxapp/avalanchego/snow/consensus/avalanche"
	"github.com/liraxapp/avalanchego/snow/consensus/snowball"
	"github.com/liraxapp/avalanchego/snow/engine/avalanche/bootstrap"
	"github.com/liraxapp/avalanchego/snow/engine/avalanche/vertex"
	"github.com/liraxapp/avalanchego/snow/engine/common"
	"github.com/liraxapp/avalanchego/snow/engine/common/queue"
)

func DefaultConfig() Config {
	vtxBlocked, _ := queue.New(memdb.New())
	txBlocked, _ := queue.New(memdb.New())
	return Config{
		Config: bootstrap.Config{
			Config:     common.DefaultConfigTest(),
			VtxBlocked: vtxBlocked,
			TxBlocked:  txBlocked,
			Manager:    &vertex.TestManager{},
			VM:         &vertex.TestVM{},
		},
		Params: avalanche.Parameters{
			Parameters: snowball.Parameters{
				Metrics:           prometheus.NewRegistry(),
				K:                 1,
				Alpha:             1,
				BetaVirtuous:      1,
				BetaRogue:         2,
				ConcurrentRepolls: 1,
			},
			Parents:   2,
			BatchSize: 1,
		},
		Consensus: &avalanche.Topological{},
	}
}
