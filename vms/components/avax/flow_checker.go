// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avax

import (
	"errors"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/utils/math"
	"github.com/liraxapp/avalanchego/utils/wrappers"
)

var (
	errInsufficientFunds = errors.New("insufficient funds")
)

// FlowChecker ...
type FlowChecker struct {
	consumed, produced map[ids.ID]uint64
	errs               wrappers.Errs
}

// NewFlowChecker ...
func NewFlowChecker() *FlowChecker {
	return &FlowChecker{
		consumed: make(map[ids.ID]uint64),
		produced: make(map[ids.ID]uint64),
	}
}

// Consume ...
func (fc *FlowChecker) Consume(assetID ids.ID, amount uint64) { fc.add(fc.consumed, assetID, amount) }

// Produce ...
func (fc *FlowChecker) Produce(assetID ids.ID, amount uint64) { fc.add(fc.produced, assetID, amount) }

func (fc *FlowChecker) add(value map[ids.ID]uint64, assetID ids.ID, amount uint64) {
	var err error
	value[assetID], err = math.Add64(value[assetID], amount)
	fc.errs.Add(err)
}

// Verify ...
func (fc *FlowChecker) Verify() error {
	if !fc.errs.Errored() {
		for assetID, producedAssetAmount := range fc.produced {
			consumedAssetAmount := fc.consumed[assetID]
			if producedAssetAmount > consumedAssetAmount {
				fc.errs.Add(errInsufficientFunds)
				break
			}
		}
	}
	return fc.errs.Err
}
