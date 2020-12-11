// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package poll

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/utils/logging"
	"github.com/liraxapp/avalanchego/utils/wrappers"
)

func TestNewSetErrorOnMetrics(t *testing.T) {
	factory := NewNoEarlyTermFactory()
	log := logging.NoLog{}
	namespace := ""
	registerer := prometheus.NewRegistry()

	errs := wrappers.Errs{}
	errs.Add(
		registerer.Register(prometheus.NewCounter(prometheus.CounterOpts{
			Name: "polls",
		})),
		registerer.Register(prometheus.NewCounter(prometheus.CounterOpts{
			Name: "poll_duration",
		})),
	)
	if errs.Errored() {
		t.Fatal(errs.Err)
	}

	if s := NewSet(factory, log, namespace, registerer); s == nil {
		t.Fatalf("shouldn't have errored due to metrics failures")
	}
}

func TestCreateAndFinishPoll(t *testing.T) {
	factory := NewNoEarlyTermFactory()
	log := logging.NoLog{}
	namespace := ""
	registerer := prometheus.NewRegistry()
	s := NewSet(factory, log, namespace, registerer)

	vtxID := ids.ID{1}
	votes := []ids.ID{vtxID}

	vdr1 := ids.NewShortID([20]byte{1})
	vdr2 := ids.NewShortID([20]byte{2}) // k = 2

	vdrs := ids.ShortBag{}
	vdrs.Add(
		vdr1,
		vdr2,
	)

	if s.Len() != 0 {
		t.Fatalf("Shouldn't have any active polls yet")
	} else if !s.Add(0, vdrs) {
		t.Fatalf("Should have been able to add a new poll")
	} else if s.Len() != 1 {
		t.Fatalf("Should only have one active poll")
	} else if s.Add(0, vdrs) {
		t.Fatalf("Shouldn't have been able to add a duplicated poll")
	} else if s.Len() != 1 {
		t.Fatalf("Should only have one active poll")
	} else if _, finished := s.Vote(1, vdr1, votes); finished {
		t.Fatalf("Shouldn't have been able to finish a non-existent poll")
	} else if _, finished := s.Vote(0, vdr1, votes); finished {
		t.Fatalf("Shouldn't have been able to finish an ongoing poll")
	} else if _, finished := s.Vote(0, vdr1, votes); finished {
		t.Fatalf("Should have dropped a duplicated poll")
	} else if result, finished := s.Vote(0, vdr2, votes); !finished {
		t.Fatalf("Should have finished the")
	} else if list := result.List(); len(list) != 1 {
		t.Fatalf("Wrong number of vertices returned")
	} else if retVtxID := list[0]; retVtxID != vtxID {
		t.Fatalf("Wrong vertex returned")
	} else if set := result.GetSet(vtxID); set.Len() != 2 {
		t.Fatalf("Wrong number of votes returned")
	}
}

func TestSetString(t *testing.T) {
	factory := NewNoEarlyTermFactory()
	log := logging.NoLog{}
	namespace := ""
	registerer := prometheus.NewRegistry()
	s := NewSet(factory, log, namespace, registerer)

	vdr1 := ids.NewShortID([20]byte{1}) // k = 1

	vdrs := ids.ShortBag{}
	vdrs.Add(vdr1)

	expected := "current polls: (Size = 1)\n" +
		"    0: waiting on Bag: (Size = 1)\n" +
		"        ID[6HgC8KRBEhXYbF4riJyJFLSHt37UNuRt]: Count = 1"
	if !s.Add(0, vdrs) {
		t.Fatalf("Should have been able to add a new poll")
	} else if str := s.String(); expected != str {
		t.Fatalf("Set return wrong string, Expected:\n%s\nReturned:\n%s",
			expected,
			str)
	}
}
