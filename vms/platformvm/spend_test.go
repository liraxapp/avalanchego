package platformvm

import (
	"testing"
	"time"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/vms/components/avax"
	"github.com/liraxapp/avalanchego/vms/components/verify"
	"github.com/liraxapp/avalanchego/vms/secp256k1fx"
)

func TestSemanticVerifySpendUTXOs(t *testing.T) {
	vm, _ := defaultVM()
	vm.Ctx.Lock.Lock()
	defer func() {
		if err := vm.Shutdown(); err != nil {
			t.Fatal(err)
		}
		vm.Ctx.Lock.Unlock()
	}()

	// The VM time during a test, unless [chainTimestamp] is set
	now := time.Unix(1607133207, 0)

	apricotPhase0Time := time.Unix(1607468400, 0)
	vm.apricotPhase0Time = apricotPhase0Time

	unsignedTx := avax.Metadata{}
	unsignedTx.Initialize([]byte{0}, []byte{1})

	// Note that setting [chainTimestamp] also set's the VM's clock.
	// Adjust input/output locktimes accordingly.
	tests := []struct {
		chainTimestamp time.Time
		description    string
		utxos          []*avax.UTXO
		ins            []*avax.TransferableInput
		outs           []*avax.TransferableOutput
		creds          []verify.Verifiable
		fee            uint64
		assetID        ids.ID
		shouldErr      bool
	}{
		{
			description: "no inputs, no outputs, no fee",
			utxos:       []*avax.UTXO{},
			ins:         []*avax.TransferableInput{},
			outs:        []*avax.TransferableOutput{},
			creds:       []verify.Verifiable{},
			fee:         0,
			assetID:     vm.Ctx.AVAXAssetID,
			shouldErr:   false,
		},
		{
			description: "no inputs, no outputs, positive fee",
			utxos:       []*avax.UTXO{},
			ins:         []*avax.TransferableInput{},
			outs:        []*avax.TransferableOutput{},
			creds:       []verify.Verifiable{},
			fee:         1,
			assetID:     vm.Ctx.AVAXAssetID,
			shouldErr:   true,
		},
		{
			description: "no inputs, no outputs, positive fee",
			utxos:       []*avax.UTXO{},
			ins:         []*avax.TransferableInput{},
			outs:        []*avax.TransferableOutput{},
			creds:       []verify.Verifiable{},
			fee:         1,
			assetID:     vm.Ctx.AVAXAssetID,
			shouldErr:   true,
		},
		{
			description: "one input, no outputs, positive fee",
			utxos: []*avax.UTXO{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*avax.TransferableInput{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*avax.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: false,
		},
		{
			description: "wrong number of credentials",
			utxos: []*avax.UTXO{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*avax.TransferableInput{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs:      []*avax.TransferableOutput{},
			creds:     []verify.Verifiable{},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: true,
		},
		{
			description: "wrong number of UTXOs",
			utxos:       []*avax.UTXO{},
			ins: []*avax.TransferableInput{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*avax.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: true,
		},
		{
			description: "invalid credential",
			utxos: []*avax.UTXO{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*avax.TransferableInput{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*avax.TransferableOutput{},
			creds: []verify.Verifiable{
				(*secp256k1fx.Credential)(nil),
			},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: true,
		},
		{
			description: "one input, no outputs, positive fee",
			utxos: []*avax.UTXO{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*avax.TransferableInput{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*avax.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: false,
		},
		{
			description: "locked one input, no outputs, no fee",
			utxos: []*avax.UTXO{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				Out: &StakeableLockOut{
					Locktime: uint64(now.Unix()) + 1,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*avax.TransferableInput{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				In: &StakeableLockIn{
					Locktime: uint64(now.Unix()) + 1,
					TransferableIn: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			}},
			outs: []*avax.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       0,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: false,
		},
		{
			description: "locked one input, no outputs, positive fee",
			utxos: []*avax.UTXO{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				Out: &StakeableLockOut{
					Locktime: uint64(now.Unix()) + 1,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*avax.TransferableInput{{
				Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
				In: &StakeableLockIn{
					Locktime: uint64(now.Unix()) + 1,
					TransferableIn: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			}},
			outs: []*avax.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: true,
		},
		{
			description: "one locked one unlock input, one locked output, positive fee",
			utxos: []*avax.UTXO{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*avax.TransferableInput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					In: &StakeableLockIn{
						Locktime: uint64(now.Unix()) + 1,
						TransferableIn: &secp256k1fx.TransferInput{
							Amt: 1,
						},
					},
				},
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*avax.TransferableOutput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: false,
		},
		{
			description: "one locked one unlock input, one locked output, positive fee, partially locked",
			utxos: []*avax.UTXO{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 2,
					},
				},
			},
			ins: []*avax.TransferableInput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					In: &StakeableLockIn{
						Locktime: uint64(now.Unix()) + 1,
						TransferableIn: &secp256k1fx.TransferInput{
							Amt: 1,
						},
					},
				},
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 2,
					},
				},
			},
			outs: []*avax.TransferableOutput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 2,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: false,
		},
		{
			chainTimestamp: apricotPhase0Time.Add(-1 * time.Second),
			description:    "one unlock input, one locked output, zero fee, unlocked, before apricot phase 0",
			utxos: []*avax.UTXO{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(apricotPhase0Time.Add(-1*time.Second).Unix()) - 1, // lock expired
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			ins: []*avax.TransferableInput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*avax.TransferableOutput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       0,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: true,
		},
		{
			chainTimestamp: apricotPhase0Time,
			description:    "one unlock input, one locked output, zero fee, unlocked, after apricot phase 0",
			utxos: []*avax.UTXO{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(apricotPhase0Time.Unix()) - 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			ins: []*avax.TransferableInput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*avax.TransferableOutput{
				{
					Asset: avax.Asset{ID: vm.Ctx.AVAXAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       0,
			assetID:   vm.Ctx.AVAXAssetID,
			shouldErr: false,
		},
	}

	for _, test := range tests {
		vm.clock.Set(now)

		if !test.chainTimestamp.IsZero() {
			if err := vm.putTimestamp(vm.DB, test.chainTimestamp); err != nil {
				t.Fatal(err)
			}
			vm.clock.Set(test.chainTimestamp)
		}

		t.Run(test.description, func(t *testing.T) {
			err := vm.semanticVerifySpendUTXOs(
				&unsignedTx,
				test.utxos,
				test.ins,
				test.outs,
				test.creds,
				test.fee,
				test.assetID,
			)

			if err == nil && test.shouldErr {
				t.Fatalf("expected error but got none")
			} else if err != nil && !test.shouldErr {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
