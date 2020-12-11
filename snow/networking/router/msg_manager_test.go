// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package router

import (
	"testing"
	"time"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow/networking/tracker"
	"github.com/liraxapp/avalanchego/snow/validators"
	"github.com/liraxapp/avalanchego/utils/logging"
	"github.com/liraxapp/avalanchego/utils/uptime"
)

func TestAddPending(t *testing.T) {
	bufferSize := 8
	vdrList := make([]validators.Validator, 0, bufferSize)
	for i := 0; i < bufferSize; i++ {
		vdr := validators.GenerateRandomValidator(2)
		vdrList = append(vdrList, vdr)
	}
	nonStakerID := ids.NewShortID([20]byte{16})

	cpuTracker := tracker.NewCPUTracker(uptime.IntervalFactory{}, time.Second)
	msgTracker := tracker.NewMessageTracker()
	vdrs := validators.NewSet()
	if err := vdrs.Set(vdrList); err != nil {
		t.Fatal(err)
	}
	resourceManager := NewMsgManager(
		vdrs,
		logging.NoLog{},
		msgTracker,
		cpuTracker,
		uint32(bufferSize),
		1,   // Allow each peer to take at most one message from pool
		0.5, // Allot half of message queue to stakers
		0.5, // Allot half of CPU time to stakers
	)

	for i, vdr := range vdrList {
		if success := resourceManager.AddPending(vdr.ID()); !success {
			t.Fatalf("Failed to take message %d.", i)
		}
	}

	if success := resourceManager.AddPending(nonStakerID); success {
		t.Fatal("Should have throttled message from non-staker when the message pool was empty")
	}

	for _, vdr := range vdrList {
		resourceManager.RemovePending(vdr.ID())
	}

	// Ensure that space is freed up after returning the messages
	// to the resource manager
	if success := resourceManager.AddPending(nonStakerID); !success {
		t.Fatal("Failed to take additional message after all previous messages were returned.")
	}
}

func TestStakerGetsThrottled(t *testing.T) {
	bufferSize := 8
	vdrList := make([]validators.Validator, 0, bufferSize)
	for i := 0; i < bufferSize; i++ {
		vdr := validators.GenerateRandomValidator(2)
		vdrList = append(vdrList, vdr)
	}

	cpuTracker := tracker.NewCPUTracker(uptime.IntervalFactory{}, time.Second)
	msgTracker := tracker.NewMessageTracker()
	vdrs := validators.NewSet()
	if err := vdrs.Set(vdrList); err != nil {
		t.Fatal(err)
	}
	resourceManager := NewMsgManager(
		vdrs,
		logging.NoLog{},
		msgTracker,
		cpuTracker,
		uint32(bufferSize),
		1,   // Allow each peer to take at most one message from pool
		0.5, // Allot half of message queue to stakers
		0.5, // Allot half of CPU time to stakers
	)

	// Ensure that a staker with only part of the stake
	// cannot take up the entire message queue
	vdrID := vdrList[0].ID()
	for i := 0; i < bufferSize; i++ {
		if success := resourceManager.AddPending(vdrID); !success {
			// The staker was throttled before taking up the whole message queue
			return
		}
	}
	t.Fatal("Staker should have been throttled before taking up the entire message queue.")
}

type infiniteResourceManager struct{}

func (i *infiniteResourceManager) AddPending(vdr ids.ShortID) bool { return true }

func (i *infiniteResourceManager) RemovePending(vdr ids.ShortID) {}

func (i *infiniteResourceManager) Utilization(vdr ids.ShortID) float64 { return 0 }

func newInfiniteResourceManager() MsgManager {
	return &infiniteResourceManager{}
}

type noResourcesManager struct{}

func (no *noResourcesManager) AddPending(vdr ids.ShortID) bool { return false }

func (no *noResourcesManager) RemovePending(vdr ids.ShortID) {}

func (no *noResourcesManager) Utilization(vdr ids.ShortID) float64 { return 1.0 }

func newNoResourcesManager() MsgManager {
	return &noResourcesManager{}
}
