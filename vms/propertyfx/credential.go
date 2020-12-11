package propertyfx

import (
	"github.com/liraxapp/avalanchego/vms/secp256k1fx"
)

// Credential ...
type Credential struct {
	secp256k1fx.Credential `serialize:"true"`
}
