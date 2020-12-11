package platformvm

import (
	"fmt"

	"github.com/liraxapp/avalanchego/snow"
	"github.com/liraxapp/avalanchego/utils/codec"
	"github.com/liraxapp/avalanchego/vms/components/avax"
)

// BaseTx contains fields common to many transaction types. It should be
// embedded in transaction implementations.
type BaseTx struct {
	avax.BaseTx `serialize:"true" json:"inputs"`

	// true iff this transaction has already passed syntactic verification
	syntacticallyVerified bool
}

// Verify returns nil iff this tx is well formed
func (tx *BaseTx) Verify(ctx *snow.Context, c codec.Manager) error {
	switch {
	case tx == nil:
		return errNilTx
	case tx.syntacticallyVerified: // already passed syntactic verification
		return nil
	}
	if err := tx.MetadataVerify(ctx); err != nil {
		return fmt.Errorf("metadata failed verification: %w", err)
	}
	for _, out := range tx.Outs {
		if err := out.Verify(); err != nil {
			return fmt.Errorf("output failed verification: %w", err)
		}
	}
	for _, in := range tx.Ins {
		if err := in.Verify(); err != nil {
			return fmt.Errorf("input failed verification: %w", err)
		}
	}
	switch {
	case !avax.IsSortedTransferableOutputs(tx.Outs, c):
		return errOutputsNotSorted
	case !avax.IsSortedAndUniqueTransferableInputs(tx.Ins):
		return errInputsNotSortedUnique
	default:
		return nil
	}
}
