package benchlist

import (
	"errors"
	"sync"
	"time"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow"
	"github.com/liraxapp/avalanchego/snow/validators"
	"github.com/liraxapp/avalanchego/utils/constants"
)

var (
	errUnknownValidators = errors.New("unknown validator set for provided chain")
)

// Manager provides an interface for a benchlist to register whether
// queries have been successful or unsuccessful and place validators with
// consistently failing queries on a benchlist to prevent waiting up to
// the full network timeout for their responses.
type Manager interface {
	// RegisterQuery registers a sent query and returns whether the query is subject to benchlist
	RegisterQuery(ids.ID, ids.ShortID, uint32, constants.MsgType) bool
	// RegisterResponse registers the response to a query message
	RegisterResponse(ids.ID, ids.ShortID, uint32)
	// QueryFailed registers that a query did not receive a response within our synchrony bound
	QueryFailed(ids.ID, ids.ShortID, uint32)
	// RegisterChain registers a new chain with metrics under [namespac]
	RegisterChain(*snow.Context, string) error
}

// Config defines the configuration for a benchlist
type Config struct {
	Validators             validators.Manager
	Threshold              int
	MinimumFailingDuration time.Duration
	Duration               time.Duration
	MaxPortion             float64
	PeerSummaryEnabled     bool
}

type benchlistManager struct {
	config *Config
	// Chain ID --> benchlist for that chain
	chainBenchlists map[ids.ID]QueryBenchlist

	lock sync.RWMutex
}

// NewManager returns a manager for chain-specific query benchlisting
func NewManager(config *Config) Manager {
	return &benchlistManager{
		config:          config,
		chainBenchlists: make(map[ids.ID]QueryBenchlist),
	}
}

func (bm *benchlistManager) RegisterChain(ctx *snow.Context, namespace string) error {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	if _, exists := bm.chainBenchlists[ctx.ChainID]; exists {
		return nil
	}

	vdrs, ok := bm.config.Validators.GetValidators(ctx.SubnetID)
	if !ok {
		return errUnknownValidators
	}

	benchlist, err := NewQueryBenchlist(
		vdrs,
		ctx,
		bm.config.Threshold,
		bm.config.MinimumFailingDuration,
		bm.config.Duration,
		bm.config.MaxPortion,
		bm.config.PeerSummaryEnabled,
		namespace,
	)
	if err != nil {
		return err
	}

	bm.chainBenchlists[ctx.ChainID] = benchlist
	return nil
}

// RegisterQuery implements the Manager interface
func (bm *benchlistManager) RegisterQuery(
	chainID ids.ID,
	validatorID ids.ShortID,
	requestID uint32,
	msgType constants.MsgType,
) bool {
	bm.lock.RLock()
	defer bm.lock.RUnlock()

	chain, exists := bm.chainBenchlists[chainID]
	if !exists {
		return false
	}

	return chain.RegisterQuery(validatorID, requestID, msgType)
}

// RegisterResponse implements the Manager interface
func (bm *benchlistManager) RegisterResponse(chainID ids.ID, validatorID ids.ShortID, requestID uint32) {
	bm.lock.RLock()
	defer bm.lock.RUnlock()

	chain, exists := bm.chainBenchlists[chainID]
	if !exists {
		return
	}

	chain.RegisterResponse(validatorID, requestID)
}

// QueryFailed implements the Manager interface
func (bm *benchlistManager) QueryFailed(chainID ids.ID, validatorID ids.ShortID, requestID uint32) {
	bm.lock.RLock()
	defer bm.lock.RUnlock()

	chain, exists := bm.chainBenchlists[chainID]
	if !exists {
		return
	}

	chain.QueryFailed(validatorID, requestID)
}

type noBenchlist struct{}

// NewNoBenchlist returns an empty benchlist that will never stop any queries
func NewNoBenchlist() Manager { return &noBenchlist{} }

func (noBenchlist) RegisterChain(*snow.Context, string) error                         { return nil }
func (noBenchlist) RegisterQuery(ids.ID, ids.ShortID, uint32, constants.MsgType) bool { return true }
func (noBenchlist) RegisterResponse(ids.ID, ids.ShortID, uint32)                      {}
func (noBenchlist) QueryFailed(ids.ID, ids.ShortID, uint32)                           {}
