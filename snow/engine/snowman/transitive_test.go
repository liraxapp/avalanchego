// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package snowman

import (
	"bytes"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow/choices"
	"github.com/liraxapp/avalanchego/snow/consensus/snowball"
	"github.com/liraxapp/avalanchego/snow/consensus/snowman"
	"github.com/liraxapp/avalanchego/snow/engine/common"
	"github.com/liraxapp/avalanchego/snow/engine/snowman/block"
	"github.com/liraxapp/avalanchego/snow/validators"
	"github.com/liraxapp/avalanchego/utils/wrappers"
)

var (
	errUnknownBlock = errors.New("unknown block")
	errUnknownBytes = errors.New("unknown bytes")

	Genesis = ids.GenerateTestID()
)

func setup(t *testing.T) (ids.ShortID, validators.Set, *common.SenderTest, *block.TestVM, *Transitive, snowman.Block) {
	config := DefaultConfig()

	vals := validators.NewSet()
	config.Validators = vals

	vdr := ids.GenerateTestShortID()
	if err := vals.AddWeight(vdr, 1); err != nil {
		t.Fatal(err)
	}

	sender := &common.SenderTest{}
	sender.T = t
	config.Sender = sender

	sender.Default(true)

	vm := &block.TestVM{}
	vm.T = t
	config.VM = vm

	vm.Default(true)
	vm.CantSetPreference = false

	gBlk := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     Genesis,
		StatusV: choices.Accepted,
	}}

	vm.LastAcceptedF = gBlk.ID
	sender.CantGetAcceptedFrontier = false

	vm.CantBootstrapping = false
	vm.CantBootstrapped = false

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if blkID != gBlk.ID() {
			t.Fatalf("Wrong block requested")
		}
		return gBlk, nil
	}

	te := &Transitive{}
	if err := te.Initialize(config); err != nil {
		t.Fatal(err)
	}

	vm.CantBootstrapping = true
	vm.CantBootstrapped = true

	vm.GetBlockF = nil
	vm.LastAcceptedF = nil
	sender.CantGetAcceptedFrontier = true

	return vdr, vals, sender, vm, te, gBlk
}

func TestEngineShutdown(t *testing.T) {
	_, _, _, vm, transitive, _ := setup(t)
	vmShutdownCalled := false
	vm.ShutdownF = func() error { vmShutdownCalled = true; return nil }
	vm.CantShutdown = false
	if err := transitive.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if !vmShutdownCalled {
		t.Fatal("Shutting down the Transitive did not shutdown the VM")
	}
}

func TestEngineAdd(t *testing.T) {
	vdr, _, sender, vm, te, _ := setup(t)

	if te.Ctx.ChainID != ids.Empty {
		t.Fatalf("Wrong chain ID")
	}

	parent := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     ids.GenerateTestID(),
		StatusV: choices.Unknown,
	}}
	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: parent,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	asked := new(bool)
	reqID := new(uint32)
	sender.GetF = func(inVdr ids.ShortID, requestID uint32, blkID ids.ID) {
		*reqID = requestID
		if *asked {
			t.Fatalf("Asked multiple times")
		}
		*asked = true
		if !vdr.Equals(inVdr) {
			t.Fatalf("Asking wrong validator for block")
		}
		if blkID != blk.Parent().ID() {
			t.Fatalf("Asking for wrong block")
		}
	}

	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if !bytes.Equal(b, blk.Bytes()) {
			t.Fatalf("Wrong bytes")
		}
		return blk, nil
	}

	if err := te.Put(vdr, 0, blk.ID(), blk.Bytes()); err != nil {
		t.Fatal(err)
	}

	vm.ParseBlockF = nil

	if !*asked {
		t.Fatalf("Didn't ask for a missing block")
	}

	if len(te.blocked) != 1 {
		t.Fatalf("Should have been blocking on request")
	}

	vm.ParseBlockF = func(b []byte) (snowman.Block, error) { return nil, errUnknownBytes }

	if err := te.Put(vdr, *reqID, blk.Parent().ID(), nil); err != nil {
		t.Fatal(err)
	}

	vm.ParseBlockF = nil

	if len(te.blocked) != 0 {
		t.Fatalf("Should have finished blocking issue")
	}
}

func TestEngineQuery(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	blocked := new(bool)
	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if *blocked {
			t.Fatalf("Sent multiple requests")
		}
		*blocked = true
		if blkID != blk.ID() {
			t.Fatalf("Wrong block requested")
		}
		return nil, errUnknownBlock
	}

	asked := new(bool)
	getRequestID := new(uint32)
	sender.GetF = func(inVdr ids.ShortID, requestID uint32, blkID ids.ID) {
		if *asked {
			t.Fatalf("Asked multiple times")
		}
		*asked = true
		*getRequestID = requestID
		if !vdr.Equals(inVdr) {
			t.Fatalf("Asking wrong validator for block")
		}
		if blk.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}

	if err := te.PullQuery(vdr, 15, blk.ID()); err != nil {
		t.Fatal(err)
	}
	if !*blocked {
		t.Fatalf("Didn't request block")
	}
	if !*asked {
		t.Fatalf("Didn't request block from validator")
	}

	queried := new(bool)
	queryRequestID := new(uint32)
	sender.PushQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		*queryRequestID = requestID
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blk.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}

	chitted := new(bool)
	sender.ChitsF = func(inVdr ids.ShortID, requestID uint32, prefSet []ids.ID) {
		if *chitted {
			t.Fatalf("Sent multiple chits")
		}
		*chitted = true
		if requestID != 15 {
			t.Fatalf("Wrong request ID")
		}
		if len(prefSet) != 1 {
			t.Fatal("Should only be one vote")
		}
		if blk.ID() != prefSet[0] {
			t.Fatalf("Wrong chits block")
		}
	}

	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if !bytes.Equal(b, blk.Bytes()) {
			t.Fatalf("Wrong bytes")
		}
		return blk, nil
	}
	if err := te.Put(vdr, *getRequestID, blk.ID(), blk.Bytes()); err != nil {
		t.Fatal(err)
	}
	vm.ParseBlockF = nil

	if !*queried {
		t.Fatalf("Didn't ask for preferences")
	}
	if !*chitted {
		t.Fatalf("Didn't provide preferences")
	}

	blk1 := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: blk,
		HeightV: 2,
		BytesV:  []byte{5, 4, 3, 2, 1, 9},
	}

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		switch blkID {
		case blk.ID():
			return blk, nil
		case blk1.ID():
			return nil, errUnknownBlock
		}
		t.Fatalf("Wrong block requested")
		panic("Should have failed")
	}

	*asked = false
	sender.GetF = func(inVdr ids.ShortID, requestID uint32, blkID ids.ID) {
		if *asked {
			t.Fatalf("Asked multiple times")
		}
		*asked = true
		*getRequestID = requestID
		if !vdr.Equals(inVdr) {
			t.Fatalf("Asking wrong validator for block")
		}
		if blk1.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}
	if err := te.Chits(vdr, *queryRequestID, []ids.ID{blk1.ID()}); err != nil {
		t.Fatal(err)
	}

	*queried = false
	sender.PushQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		*queryRequestID = requestID
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blkID != blk1.ID() {
			t.Fatalf("Asking for wrong block")
		}
	}

	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if !bytes.Equal(b, blk1.Bytes()) {
			t.Fatalf("Wrong bytes")
		}

		vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
			switch blkID {
			case blk.ID():
				return blk, nil
			case blk1.ID():
				return blk1, nil
			}
			t.Fatalf("Wrong block requested")
			panic("Should have failed")
		}

		return blk1, nil
	}
	if err := te.Put(vdr, *getRequestID, blk1.ID(), blk1.Bytes()); err != nil {
		t.Fatal(err)
	}
	vm.ParseBlockF = nil

	if blk1.Status() != choices.Accepted {
		t.Fatalf("Should have executed block")
	}
	if len(te.blocked) != 0 {
		t.Fatalf("Should have finished blocking")
	}

	_ = te.polls.String() // Shouldn't panic

	if err := te.QueryFailed(vdr, *queryRequestID); err != nil {
		t.Fatal(err)
	}
	if len(te.blocked) != 0 {
		t.Fatalf("Should have finished blocking")
	}
}

func TestEngineMultipleQuery(t *testing.T) {
	config := DefaultConfig()

	config.Params = snowball.Parameters{
		Metrics:           prometheus.NewRegistry(),
		K:                 3,
		Alpha:             2,
		BetaVirtuous:      1,
		BetaRogue:         2,
		ConcurrentRepolls: 1,
	}

	vals := validators.NewSet()
	config.Validators = vals

	vdr0 := ids.GenerateTestShortID()
	vdr1 := ids.GenerateTestShortID()
	vdr2 := ids.GenerateTestShortID()

	errs := wrappers.Errs{}
	errs.Add(
		vals.AddWeight(vdr0, 1),
		vals.AddWeight(vdr1, 1),
		vals.AddWeight(vdr2, 1),
	)
	if errs.Errored() {
		t.Fatal(errs.Err)
	}

	sender := &common.SenderTest{}
	sender.T = t
	config.Sender = sender

	sender.Default(true)

	vm := &block.TestVM{}
	vm.T = t
	config.VM = vm

	vm.Default(true)
	vm.CantSetPreference = false

	gBlk := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     ids.GenerateTestID(),
		StatusV: choices.Accepted,
	}}

	vm.LastAcceptedF = gBlk.ID
	sender.CantGetAcceptedFrontier = false

	vm.CantBootstrapping = false
	vm.CantBootstrapped = false
	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if blkID != gBlk.ID() {
			t.Fatalf("Wrong block requested")
		}
		return gBlk, nil
	}

	te := &Transitive{}
	if err := te.Initialize(config); err != nil {
		t.Fatal(err)
	}

	vm.CantBootstrapping = true
	vm.CantBootstrapped = true

	vm.GetBlockF = nil
	vm.LastAcceptedF = nil
	sender.CantGetAcceptedFrontier = true

	blk0 := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	queried := new(bool)
	queryRequestID := new(uint32)
	sender.PushQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		*queryRequestID = requestID
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr0, vdr1, vdr2)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blk0.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}

	if err := te.issue(blk0); err != nil {
		t.Fatal(err)
	}

	blk1 := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: blk0,
		HeightV: 2,
		BytesV:  []byte{2},
	}

	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		switch id {
		case gBlk.ID():
			return gBlk, nil
		case blk0.ID():
			return blk0, nil
		case blk1.ID():
			return nil, errUnknownBlock
		}
		t.Fatalf("Unknown block")
		panic("Should have errored")
	}

	asked := new(bool)
	getRequestID := new(uint32)
	sender.GetF = func(inVdr ids.ShortID, requestID uint32, blkID ids.ID) {
		if *asked {
			t.Fatalf("Asked multiple times")
		}
		*asked = true
		*getRequestID = requestID
		if !vdr0.Equals(inVdr) {
			t.Fatalf("Asking wrong validator for block")
		}
		if blk1.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}
	blkSet := []ids.ID{blk1.ID()}
	if err := te.Chits(vdr0, *queryRequestID, blkSet); err != nil {
		t.Fatal(err)
	}
	if err := te.Chits(vdr1, *queryRequestID, blkSet); err != nil {
		t.Fatal(err)
	}

	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
			switch {
			case blkID == blk0.ID():
				return blk0, nil
			case blkID == blk1.ID():
				return blk1, nil
			}
			t.Fatalf("Wrong block requested")
			panic("Should have failed")
		}

		return blk1, nil
	}

	*queried = false
	secondQueryRequestID := new(uint32)
	sender.PushQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		*secondQueryRequestID = requestID
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr0, vdr1, vdr2)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blk1.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}
	if err := te.Put(vdr0, *getRequestID, blk1.ID(), blk1.Bytes()); err != nil {
		t.Fatal(err)
	}

	// Should be dropped because the query was already filled
	blkSet = []ids.ID{blk0.ID()}
	if err := te.Chits(vdr2, *queryRequestID, blkSet); err != nil {
		t.Fatal(err)
	}

	if blk1.Status() != choices.Accepted {
		t.Fatalf("Should have executed block")
	}
	if len(te.blocked) != 0 {
		t.Fatalf("Should have finished blocking")
	}
}

func TestEngineBlockedIssue(t *testing.T) {
	_, _, sender, _, te, gBlk := setup(t)

	sender.Default(false)

	blk0 := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Unknown,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}
	blk1 := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: blk0,
		HeightV: 2,
		BytesV:  []byte{2},
	}

	if err := te.issue(blk1); err != nil {
		t.Fatal(err)
	}

	blk0.StatusV = choices.Processing
	if err := te.issue(blk0); err != nil {
		t.Fatal(err)
	}

	if blk1.ID() != te.Consensus.Preference() {
		t.Fatalf("Should have issued blk1")
	}
}

func TestEngineAbandonResponse(t *testing.T) {
	vdr, _, sender, _, te, gBlk := setup(t)

	sender.Default(false)

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Unknown,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	if err := te.issue(blk); err != nil {
		t.Fatal(err)
	}
	if err := te.QueryFailed(vdr, 1); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 0 {
		t.Fatalf("Should have removed blocking event")
	}
}

func TestEngineFetchBlock(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(false)

	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		if id == gBlk.ID() {
			return gBlk, nil
		}
		t.Fatalf("Unknown block")
		panic("Should have failed")
	}

	added := new(bool)
	sender.PutF = func(inVdr ids.ShortID, requestID uint32, blkID ids.ID, blk []byte) {
		if !vdr.Equals(inVdr) {
			t.Fatalf("Wrong validator")
		}
		if requestID != 123 {
			t.Fatalf("Wrong request id")
		}
		if gBlk.ID() != blkID {
			t.Fatalf("Wrong blockID")
		}
		*added = true
	}

	if err := te.Get(vdr, 123, gBlk.ID()); err != nil {
		t.Fatal(err)
	}

	if !*added {
		t.Fatalf("Should have sent block to peer")
	}
}

func TestEnginePushQuery(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if bytes.Equal(b, blk.Bytes()) {
			return blk, nil
		}
		return nil, errUnknownBytes
	}

	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		if id == blk.ID() {
			return blk, nil
		}
		t.Fatal(errUnknownBytes)
		panic(errUnknownBytes)
	}

	chitted := new(bool)
	sender.ChitsF = func(inVdr ids.ShortID, requestID uint32, votes []ids.ID) {
		if *chitted {
			t.Fatalf("Sent chit multiple times")
		}
		*chitted = true
		if !inVdr.Equals(vdr) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if requestID != 20 {
			t.Fatalf("Wrong request id")
		}
		if len(votes) != 1 {
			t.Fatal("votes should only have one element")
		}
		if blk.ID() != votes[0] {
			t.Fatalf("Asking for wrong block")
		}
	}

	queried := new(bool)
	sender.PushQueryF = func(inVdrs ids.ShortSet, _ uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blk.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}

	if err := te.PushQuery(vdr, 20, blk.ID(), blk.Bytes()); err != nil {
		t.Fatal(err)
	}

	if !*chitted {
		t.Fatalf("Should have sent a chit to the peer")
	}
	if !*queried {
		t.Fatalf("Should have sent a query to the peer")
	}
}

func TestEngineBuildBlock(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	queried := new(bool)
	sender.PushQueryF = func(inVdrs ids.ShortSet, _ uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
	}

	vm.BuildBlockF = func() (snowman.Block, error) { return blk, nil }
	if err := te.Notify(common.PendingTxs); err != nil {
		t.Fatal(err)
	}

	if !*queried {
		t.Fatalf("Should have sent a query to the peer")
	}
}

func TestEngineRepoll(t *testing.T) {
	vdr, _, sender, _, te, _ := setup(t)

	sender.Default(true)

	queried := new(bool)
	sender.PullQueryF = func(inVdrs ids.ShortSet, _ uint32, blkID ids.ID) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
	}

	te.repoll()

	if !*queried {
		t.Fatalf("Should have sent a query to the peer")
	}
}

func TestVoteCanceling(t *testing.T) {
	config := DefaultConfig()

	config.Params = snowball.Parameters{
		Metrics:           prometheus.NewRegistry(),
		K:                 3,
		Alpha:             2,
		BetaVirtuous:      1,
		BetaRogue:         2,
		ConcurrentRepolls: 1,
	}

	vals := validators.NewSet()
	config.Validators = vals

	vdr0 := ids.GenerateTestShortID()
	vdr1 := ids.GenerateTestShortID()
	vdr2 := ids.GenerateTestShortID()

	errs := wrappers.Errs{}
	errs.Add(
		vals.AddWeight(vdr0, 1),
		vals.AddWeight(vdr1, 1),
		vals.AddWeight(vdr2, 1),
	)
	if errs.Errored() {
		t.Fatal(errs.Err)
	}

	sender := &common.SenderTest{}
	sender.T = t
	config.Sender = sender

	sender.Default(true)

	vm := &block.TestVM{}
	vm.T = t
	config.VM = vm

	vm.Default(true)
	vm.CantSetPreference = false

	gBlk := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     ids.GenerateTestID(),
		StatusV: choices.Accepted,
	}}

	vm.LastAcceptedF = gBlk.ID
	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		switch id {
		case gBlk.ID():
			return gBlk, nil
		default:
			t.Fatalf("Loaded unknown block")
			panic("Should have failed")
		}
	}
	sender.CantGetAcceptedFrontier = false

	vm.CantBootstrapping = false
	vm.CantBootstrapped = false

	te := &Transitive{}
	if err := te.Initialize(config); err != nil {
		t.Fatal(err)
	}

	vm.CantBootstrapping = false
	vm.CantBootstrapped = false

	vm.LastAcceptedF = nil
	sender.CantGetAcceptedFrontier = true

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	queried := new(bool)
	queryRequestID := new(uint32)
	sender.PushQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		*queryRequestID = requestID
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr0, vdr1, vdr2)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blk.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}

	if err := te.issue(blk); err != nil {
		t.Fatal(err)
	}

	if te.polls.Len() != 1 {
		t.Fatalf("Shouldn't have finished blocking issue")
	}

	if err := te.QueryFailed(vdr0, *queryRequestID); err != nil {
		t.Fatal(err)
	}

	if te.polls.Len() != 1 {
		t.Fatalf("Shouldn't have finished blocking issue")
	}

	repolled := new(bool)
	sender.PullQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID) {
		*repolled = true
	}
	if err := te.QueryFailed(vdr1, *queryRequestID); err != nil {
		t.Fatal(err)
	}

	if !*repolled {
		t.Fatalf("Should have finished blocking issue and repolled the network")
	}
}

func TestEngineNoQuery(t *testing.T) {
	config := DefaultConfig()

	sender := &common.SenderTest{}
	sender.T = t
	config.Sender = sender

	sender.Default(true)
	sender.CantGetAcceptedFrontier = false

	gBlk := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     ids.GenerateTestID(),
		StatusV: choices.Accepted,
	}}

	vm := &block.TestVM{}
	vm.T = t
	vm.LastAcceptedF = gBlk.ID

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if blkID == gBlk.ID() {
			return gBlk, nil
		}
		return nil, errUnknownBlock
	}

	config.VM = vm
	te := &Transitive{}
	if err := te.Initialize(config); err != nil {
		t.Fatal(err)
	}

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	if err := te.issue(blk); err != nil {
		t.Fatal(err)
	}
}

func TestEngineNoRepollQuery(t *testing.T) {
	config := DefaultConfig()

	sender := &common.SenderTest{}
	sender.T = t
	config.Sender = sender

	sender.Default(true)
	sender.CantGetAcceptedFrontier = false

	gBlk := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     ids.GenerateTestID(),
		StatusV: choices.Accepted,
	}}

	vm := &block.TestVM{}
	vm.T = t
	vm.LastAcceptedF = gBlk.ID

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if blkID == gBlk.ID() {
			return gBlk, nil
		}
		return nil, errUnknownBlock
	}

	config.VM = vm
	te := &Transitive{}
	if err := te.Initialize(config); err != nil {
		t.Fatal(err)
	}

	te.repoll()
}

func TestEngineAbandonQuery(t *testing.T) {
	vdr, _, sender, vm, te, _ := setup(t)

	sender.Default(true)

	blkID := ids.GenerateTestID()

	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		switch id {
		case blkID:
			return nil, errUnknownBlock
		default:
			t.Fatalf("Loaded unknown block")
			panic("Should have failed")
		}
	}

	reqID := new(uint32)
	sender.GetF = func(_ ids.ShortID, requestID uint32, _ ids.ID) {
		*reqID = requestID
	}

	if err := te.PullQuery(vdr, 0, blkID); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 1 {
		t.Fatalf("Should have blocked on request")
	}

	if err := te.GetFailed(vdr, *reqID); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 0 {
		t.Fatalf("Should have removed request")
	}
}

func TestEngineAbandonChit(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	sender.CantPushQuery = false

	if err := te.issue(blk); err != nil {
		t.Fatal(err)
	}

	fakeBlkID := ids.GenerateTestID()
	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		switch id {
		case fakeBlkID:
			return nil, errUnknownBlock
		default:
			t.Fatalf("Loaded unknown block")
			panic("Should have failed")
		}
	}

	reqID := new(uint32)
	sender.GetF = func(_ ids.ShortID, requestID uint32, _ ids.ID) {
		*reqID = requestID
	}

	if err := te.Chits(vdr, 0, []ids.ID{fakeBlkID}); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 1 {
		t.Fatalf("Should have blocked on request")
	}

	if err := te.GetFailed(vdr, *reqID); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 0 {
		t.Fatalf("Should have removed request")
	}
}

func TestEngineBlockingChitRequest(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	missingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Unknown,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}
	parentBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: missingBlk,
		HeightV: 2,
		BytesV:  []byte{2},
	}
	blockingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: parentBlk,
		HeightV: 3,
		BytesV:  []byte{3},
	}

	if err := te.issue(parentBlk); err != nil {
		t.Fatal(err)
	}

	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		switch {
		case bytes.Equal(b, blockingBlk.Bytes()):
			return blockingBlk, nil
		default:
			t.Fatalf("Loaded unknown block")
			panic("Should have failed")
		}
	}
	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		switch {
		case blkID == blockingBlk.ID():
			return blockingBlk, nil
		default:
			t.Fatalf("Loaded unknown block")
			panic("Should have failed")
		}
	}

	if err := te.PushQuery(vdr, 0, blockingBlk.ID(), blockingBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 3 {
		t.Fatalf("Both inserts should be blocking in addition to the chit request")
	}

	sender.CantPushQuery = false
	sender.CantChits = false

	missingBlk.StatusV = choices.Processing
	if err := te.issue(missingBlk); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 0 {
		t.Fatalf("Both inserts should not longer be blocking")
	}
}

func TestEngineBlockingChitResponse(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	issuedBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}
	missingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Unknown,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{2},
	}
	blockingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: missingBlk,
		HeightV: 2,
		BytesV:  []byte{3},
	}

	if err := te.issue(blockingBlk); err != nil {
		t.Fatal(err)
	}

	queryRequestID := new(uint32)
	sender.PushQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID, blkBytes []byte) {
		*queryRequestID = requestID
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blkID != issuedBlk.ID() {
			t.Fatalf("Asking for wrong block")
		}
	}

	if err := te.issue(issuedBlk); err != nil {
		t.Fatal(err)
	}

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		switch {
		case blkID == blockingBlk.ID():
			return blockingBlk, nil
		default:
			t.Fatalf("Loaded unknown block")
			panic("Should have failed")
		}
	}
	if err := te.Chits(vdr, *queryRequestID, []ids.ID{blockingBlk.ID()}); err != nil {
		t.Fatal(err)
	}

	if len(te.blocked) != 2 {
		t.Fatalf("The insert and the chit should be blocking")
	}

	sender.PushQueryF = nil
	sender.CantPushQuery = false

	missingBlk.StatusV = choices.Processing
	if err := te.issue(missingBlk); err != nil {
		t.Fatal(err)
	}
}

func TestEngineRetryFetch(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	missingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Unknown,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	vm.CantGetBlock = false

	reqID := new(uint32)
	sender.GetF = func(_ ids.ShortID, requestID uint32, _ ids.ID) {
		*reqID = requestID
	}

	if err := te.PullQuery(vdr, 0, missingBlk.ID()); err != nil {
		t.Fatal(err)
	}

	vm.CantGetBlock = true
	sender.GetF = nil

	if err := te.GetFailed(vdr, *reqID); err != nil {
		t.Fatal(err)
	}

	vm.CantGetBlock = false

	called := new(bool)
	sender.GetF = func(ids.ShortID, uint32, ids.ID) {
		*called = true
	}

	if err := te.PullQuery(vdr, 0, missingBlk.ID()); err != nil {
		t.Fatal(err)
	}

	vm.CantGetBlock = true
	sender.GetF = nil

	if !*called {
		t.Fatalf("Should have requested the block again")
	}
}

func TestEngineUndeclaredDependencyDeadlock(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	validBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}
	invalidBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: validBlk,
		HeightV: 2,
		VerifyV: errors.New(""),
		BytesV:  []byte{2},
	}

	validBlkID := validBlk.ID()
	invalidBlkID := invalidBlk.ID()

	reqID := new(uint32)
	sender.PushQueryF = func(_ ids.ShortSet, requestID uint32, _ ids.ID, _ []byte) {
		*reqID = requestID
	}

	if err := te.issue(validBlk); err != nil {
		t.Fatal(err)
	}

	sender.PushQueryF = nil

	if err := te.issue(invalidBlk); err != nil {
		t.Fatal(err)
	}

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		switch {
		case blkID == validBlkID:
			return validBlk, nil
		case blkID == invalidBlkID:
			return invalidBlk, nil
		}
		return nil, errUnknownBlock
	}

	if err := te.Chits(vdr, *reqID, []ids.ID{invalidBlkID}); err != nil {
		t.Fatal(err)
	}

	vm.GetBlockF = nil

	if status := validBlk.Status(); status != choices.Accepted {
		t.Fatalf("Should have bubbled invalid votes to the valid parent")
	}
}

func TestEngineGossip(t *testing.T) {
	_, _, sender, vm, te, gBlk := setup(t)

	vm.LastAcceptedF = gBlk.ID
	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if blkID == gBlk.ID() {
			return gBlk, nil
		}
		t.Fatal(errUnknownBlock)
		return nil, errUnknownBlock
	}

	called := new(bool)
	sender.GossipF = func(blkID ids.ID, blkBytes []byte) {
		*called = true
		if blkID != gBlk.ID() {
			t.Fatal(errUnknownBlock)
		}
		if !bytes.Equal(blkBytes, gBlk.Bytes()) {
			t.Fatal(errUnknownBytes)
		}
	}

	if err := te.Gossip(); err != nil {
		t.Fatal(err)
	}

	if !*called {
		t.Fatalf("Should have gossiped the block")
	}
}

func TestEngineInvalidBlockIgnoredFromUnexpectedPeer(t *testing.T) {
	vdr, vdrs, sender, vm, te, gBlk := setup(t)

	secondVdr := ids.GenerateTestShortID()
	if err := vdrs.AddWeight(secondVdr, 1); err != nil {
		t.Fatal(err)
	}

	sender.Default(true)

	missingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Unknown,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}
	pendingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: missingBlk,
		HeightV: 2,
		BytesV:  []byte{2},
	}

	parsed := new(bool)
	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if bytes.Equal(b, pendingBlk.Bytes()) {
			*parsed = true
			return pendingBlk, nil
		}
		return nil, errUnknownBlock
	}

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if !*parsed {
			return nil, errUnknownBlock
		}

		if blkID == pendingBlk.ID() {
			return pendingBlk, nil
		}
		return nil, errUnknownBlock
	}

	reqID := new(uint32)
	sender.GetF = func(reqVdr ids.ShortID, requestID uint32, blkID ids.ID) {
		*reqID = requestID
		if !reqVdr.Equals(vdr) {
			t.Fatalf("Wrong validator requested")
		}
		if blkID != missingBlk.ID() {
			t.Fatalf("Wrong block requested")
		}
	}

	if err := te.PushQuery(vdr, 0, pendingBlk.ID(), pendingBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	if err := te.Put(secondVdr, *reqID, missingBlk.ID(), []byte{3}); err != nil {
		t.Fatal(err)
	}

	*parsed = false
	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if bytes.Equal(b, missingBlk.Bytes()) {
			*parsed = true
			return missingBlk, nil
		}
		return nil, errUnknownBlock
	}
	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if !*parsed {
			return nil, errUnknownBlock
		}

		if blkID == missingBlk.ID() {
			return missingBlk, nil
		}
		return nil, errUnknownBlock
	}
	sender.CantPushQuery = false
	sender.CantChits = false

	missingBlk.StatusV = choices.Processing

	if err := te.Put(vdr, *reqID, missingBlk.ID(), missingBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	pref := te.Consensus.Preference()
	if pref != pendingBlk.ID() {
		t.Fatalf("Shouldn't have abandoned the pending block")
	}
}

func TestEnginePushQueryRequestIDConflict(t *testing.T) {
	vdr, _, sender, vm, te, gBlk := setup(t)

	sender.Default(true)

	missingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Unknown,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}
	pendingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: missingBlk,
		HeightV: 2,
		BytesV:  []byte{2},
	}

	randomBlkID := ids.GenerateTestID()

	parsed := new(bool)
	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if bytes.Equal(b, pendingBlk.Bytes()) {
			*parsed = true
			return pendingBlk, nil
		}
		return nil, errUnknownBlock
	}

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if !*parsed {
			return nil, errUnknownBlock
		}

		if blkID == pendingBlk.ID() {
			return pendingBlk, nil
		}
		return nil, errUnknownBlock
	}

	reqID := new(uint32)
	sender.GetF = func(reqVdr ids.ShortID, requestID uint32, blkID ids.ID) {
		*reqID = requestID
		if !reqVdr.Equals(vdr) {
			t.Fatalf("Wrong validator requested")
		}
		if blkID != missingBlk.ID() {
			t.Fatalf("Wrong block requested")
		}
	}

	if err := te.PushQuery(vdr, 0, pendingBlk.ID(), pendingBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	sender.GetF = nil
	sender.CantGet = false

	if err := te.PushQuery(vdr, *reqID, randomBlkID, []byte{3}); err != nil {
		t.Fatal(err)
	}

	*parsed = false
	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if bytes.Equal(b, missingBlk.Bytes()) {
			*parsed = true
			return missingBlk, nil
		}
		return nil, errUnknownBlock
	}
	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if !*parsed {
			return nil, errUnknownBlock
		}

		if blkID == missingBlk.ID() {
			return missingBlk, nil
		}
		return nil, errUnknownBlock
	}
	sender.CantPushQuery = false
	sender.CantChits = false

	if err := te.Put(vdr, *reqID, missingBlk.ID(), missingBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	pref := te.Consensus.Preference()
	if pref != pendingBlk.ID() {
		t.Fatalf("Shouldn't have abandoned the pending block")
	}
}

func TestEngineAggressivePolling(t *testing.T) {
	config := DefaultConfig()

	config.Params.ConcurrentRepolls = 2

	vals := validators.NewSet()
	config.Validators = vals

	vdr := ids.GenerateTestShortID()
	if err := vals.AddWeight(vdr, 1); err != nil {
		t.Fatal(err)
	}

	sender := &common.SenderTest{}
	sender.T = t
	config.Sender = sender

	sender.Default(true)

	vm := &block.TestVM{}
	vm.T = t
	config.VM = vm

	vm.Default(true)
	vm.CantSetPreference = false

	gBlk := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     ids.GenerateTestID(),
		StatusV: choices.Accepted,
	}}

	vm.LastAcceptedF = gBlk.ID
	sender.CantGetAcceptedFrontier = false

	vm.CantBootstrapping = false
	vm.CantBootstrapped = false

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if blkID != gBlk.ID() {
			t.Fatalf("Wrong block requested")
		}
		return gBlk, nil
	}

	te := &Transitive{}
	if err := te.Initialize(config); err != nil {
		t.Fatal(err)
	}

	vm.CantBootstrapping = true
	vm.CantBootstrapped = true

	vm.GetBlockF = nil
	vm.LastAcceptedF = nil
	sender.CantGetAcceptedFrontier = true

	sender.Default(true)

	pendingBlk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	parsed := new(bool)
	vm.ParseBlockF = func(b []byte) (snowman.Block, error) {
		if bytes.Equal(b, pendingBlk.Bytes()) {
			*parsed = true
			return pendingBlk, nil
		}
		return nil, errUnknownBlock
	}

	vm.GetBlockF = func(blkID ids.ID) (snowman.Block, error) {
		if !*parsed {
			return nil, errUnknownBlock
		}

		if blkID == pendingBlk.ID() {
			return pendingBlk, nil
		}
		return nil, errUnknownBlock
	}

	numPushed := new(int)
	sender.PushQueryF = func(_ ids.ShortSet, _ uint32, _ ids.ID, _ []byte) { *numPushed++ }

	numPulled := new(int)
	sender.PullQueryF = func(_ ids.ShortSet, _ uint32, _ ids.ID) { *numPulled++ }

	if err := te.Put(vdr, 0, pendingBlk.ID(), pendingBlk.Bytes()); err != nil {
		t.Fatal(err)
	}

	if *numPushed != 1 {
		t.Fatalf("Should have initially sent a push query")
	}

	if *numPulled != 1 {
		t.Fatalf("Should have sent an additional pull query")
	}
}

func TestEngineDoubleChit(t *testing.T) {
	config := DefaultConfig()

	config.Params = snowball.Parameters{
		Metrics:      prometheus.NewRegistry(),
		K:            2,
		Alpha:        2,
		BetaVirtuous: 1,
		BetaRogue:    2,
	}

	vals := validators.NewSet()
	config.Validators = vals

	vdr0 := ids.GenerateTestShortID()
	vdr1 := ids.GenerateTestShortID()

	if err := vals.AddWeight(vdr0, 1); err != nil {
		t.Fatal(err)
	}
	if err := vals.AddWeight(vdr1, 1); err != nil {
		t.Fatal(err)
	}

	sender := &common.SenderTest{}
	sender.T = t
	config.Sender = sender

	sender.Default(true)

	vm := &block.TestVM{}
	vm.T = t
	config.VM = vm

	vm.Default(true)
	vm.CantSetPreference = false

	gBlk := &snowman.TestBlock{TestDecidable: choices.TestDecidable{
		IDV:     ids.GenerateTestID(),
		StatusV: choices.Accepted,
	}}

	vm.LastAcceptedF = gBlk.ID
	sender.CantGetAcceptedFrontier = false

	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		if id == gBlk.ID() {
			return gBlk, nil
		}
		t.Fatalf("Unknown block")
		panic("Should have errored")
	}

	vm.CantBootstrapping = false
	vm.CantBootstrapped = false

	te := &Transitive{}
	if err := te.Initialize(config); err != nil {
		t.Fatal(err)
	}

	vm.CantBootstrapping = true
	vm.CantBootstrapped = true

	vm.LastAcceptedF = nil
	sender.CantGetAcceptedFrontier = true

	blk := &snowman.TestBlock{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		ParentV: gBlk,
		HeightV: 1,
		BytesV:  []byte{1},
	}

	queried := new(bool)
	queryRequestID := new(uint32)
	sender.PushQueryF = func(inVdrs ids.ShortSet, requestID uint32, blkID ids.ID, blkBytes []byte) {
		if *queried {
			t.Fatalf("Asked multiple times")
		}
		*queried = true
		*queryRequestID = requestID
		vdrSet := ids.ShortSet{}
		vdrSet.Add(vdr0, vdr1)
		if !inVdrs.Equals(vdrSet) {
			t.Fatalf("Asking wrong validator for preference")
		}
		if blk.ID() != blkID {
			t.Fatalf("Asking for wrong block")
		}
	}

	if err := te.issue(blk); err != nil {
		t.Fatal(err)
	}

	vm.GetBlockF = func(id ids.ID) (snowman.Block, error) {
		switch id {
		case gBlk.ID():
			return gBlk, nil
		case blk.ID():
			return blk, nil
		}
		t.Fatalf("Unknown block")
		panic("Should have errored")
	}

	blkSet := []ids.ID{blk.ID()}

	if status := blk.Status(); status != choices.Processing {
		t.Fatalf("Wrong status: %s ; expected: %s", status, choices.Processing)
	}

	if err := te.Chits(vdr0, *queryRequestID, blkSet); err != nil {
		t.Fatal(err)
	}

	if status := blk.Status(); status != choices.Processing {
		t.Fatalf("Wrong status: %s ; expected: %s", status, choices.Processing)
	}

	if err := te.Chits(vdr0, *queryRequestID, blkSet); err != nil {
		t.Fatal(err)
	}

	if status := blk.Status(); status != choices.Processing {
		t.Fatalf("Wrong status: %s ; expected: %s", status, choices.Processing)
	}

	if err := te.Chits(vdr1, *queryRequestID, blkSet); err != nil {
		t.Fatal(err)
	}

	if status := blk.Status(); status != choices.Accepted {
		t.Fatalf("Wrong status: %s ; expected: %s", status, choices.Accepted)
	}
}
