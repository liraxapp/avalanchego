// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snow

import (
	"io"
	"net/http"
	"sync"

	stdatomic "sync/atomic"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/liraxapp/avalanchego/chains/atomic"
	"github.com/liraxapp/avalanchego/database"
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/utils/logging"
)

// Callable ...
type Callable interface {
	Call(writer http.ResponseWriter, method, base, endpoint string, body io.Reader, headers map[string]string) error
}

// EventDispatcher ...
type EventDispatcher interface {
	Issue(ctx *Context, containerID ids.ID, container []byte)
	Accept(ctx *Context, containerID ids.ID, container []byte)
	Reject(ctx *Context, containerID ids.ID, container []byte)
}

// Keystore ...
type Keystore interface {
	GetDatabase(username, password string) (database.Database, error)
}

// AliasLookup ...
type AliasLookup interface {
	Lookup(alias string) (ids.ID, error)
	PrimaryAlias(id ids.ID) (string, error)
}

// SubnetLookup ...
type SubnetLookup interface {
	SubnetID(chainID ids.ID) (ids.ID, error)
}

// Context is information about the current execution.
// [NetworkID] is the ID of the network this context exists within.
// [ChainID] is the ID of the chain this context exists within.
// [NodeID] is the ID of this node
type Context struct {
	NetworkID uint32
	SubnetID  ids.ID
	ChainID   ids.ID
	NodeID    ids.ShortID

	XChainID    ids.ID
	AVAXAssetID ids.ID

	Log                 logging.Logger
	DecisionDispatcher  EventDispatcher
	ConsensusDispatcher EventDispatcher
	Lock                sync.RWMutex
	Keystore            Keystore
	SharedMemory        atomic.SharedMemory
	BCLookup            AliasLookup
	SNLookup            SubnetLookup

	// Non-zero iff this chain bootstrapped. Should only be accessed atomically.
	bootstrapped uint32
	Namespace    string
	Metrics      prometheus.Registerer
}

// IsBootstrapped returns true iff this chain is done bootstrapping
func (ctx *Context) IsBootstrapped() bool {
	return stdatomic.LoadUint32(&ctx.bootstrapped) > 0
}

// Bootstrapped marks this chain as done bootstrapping
func (ctx *Context) Bootstrapped() {
	stdatomic.StoreUint32(&ctx.bootstrapped, 1)
}

// DefaultContextTest ...
func DefaultContextTest() *Context {
	aliaser := &ids.Aliaser{}
	aliaser.Initialize()
	return &Context{
		NetworkID:           0,
		SubnetID:            ids.Empty,
		ChainID:             ids.Empty,
		NodeID:              ids.ShortEmpty,
		Log:                 logging.NoLog{},
		DecisionDispatcher:  emptyEventDispatcher{},
		ConsensusDispatcher: emptyEventDispatcher{},
		BCLookup:            aliaser,
		Namespace:           "",
		Metrics:             prometheus.NewRegistry(),
	}
}

type emptyEventDispatcher struct{}

func (emptyEventDispatcher) Issue(*Context, ids.ID, []byte)  {}
func (emptyEventDispatcher) Accept(*Context, ids.ID, []byte) {}
func (emptyEventDispatcher) Reject(*Context, ids.ID, []byte) {}
