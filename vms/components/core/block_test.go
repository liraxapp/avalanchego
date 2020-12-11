package core

import (
	"testing"

	"github.com/liraxapp/avalanchego/database/memdb"
	"github.com/liraxapp/avalanchego/database/versiondb"
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow/choices"
	"github.com/liraxapp/avalanchego/snow/consensus/snowman"
)

func TestBlock(t *testing.T) {
	parentID := ids.ID{1, 2, 3, 4, 5}
	db := versiondb.New(memdb.New())
	state, err := NewSnowmanState(func([]byte) (snowman.Block, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	b := NewBlock(parentID, 1)

	b.Initialize([]byte{1, 2, 3}, &SnowmanVM{
		DB:    db,
		State: state,
	})

	if b.Hght != 1 {
		t.Fatal("height should be 1")
	}

	// should be unknown until someone queries for it
	if status := b.Metadata.status; status != choices.Unknown {
		t.Fatalf("status should be unknown but is %s", status)
	}

	// querying should change status to processing
	if status := b.Status(); status != choices.Processing {
		t.Fatalf("status should be processing but is %s", status)
	}

	if err := b.Accept(); err != nil {
		t.Fatal(err)
	}
	if status := b.Status(); status != choices.Accepted {
		t.Fatalf("status should be accepted but is %s", status)
	}

	if err := b.Reject(); err != nil {
		t.Fatal(err)
	}
	if status := b.Status(); status != choices.Rejected {
		t.Fatalf("status should be rejected but is %s", status)
	}
}
