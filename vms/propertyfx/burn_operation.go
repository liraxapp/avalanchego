package propertyfx

import (
	"github.com/liraxapp/avalanchego/vms/components/verify"
	"github.com/liraxapp/avalanchego/vms/secp256k1fx"
)

// BurnOperation ...
type BurnOperation struct {
	secp256k1fx.Input `serialize:"true"`
}

// Outs ...
func (op *BurnOperation) Outs() []verify.State { return nil }
