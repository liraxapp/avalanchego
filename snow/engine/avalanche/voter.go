// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avalanche

import (
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/snow/consensus/snowstorm"
	"github.com/liraxapp/avalanchego/snow/engine/avalanche/vertex"
)

// Voter records chits received from [vdr] once its dependencies are met.
type voter struct {
	t         *Transitive
	vdr       ids.ShortID
	requestID uint32
	response  []ids.ID
	deps      ids.Set
}

func (v *voter) Dependencies() ids.Set { return v.deps }

// Mark that a dependency has been met.
func (v *voter) Fulfill(id ids.ID) {
	v.deps.Remove(id)
	v.Update()
}

// Abandon this attempt to record chits.
func (v *voter) Abandon(id ids.ID) { v.Fulfill(id) }

func (v *voter) Update() {
	if v.deps.Len() != 0 || v.t.errs.Errored() {
		return
	}

	results, finished := v.t.polls.Vote(v.requestID, v.vdr, v.response)
	if !finished {
		return
	}
	results, err := v.bubbleVotes(results)
	if err != nil {
		v.t.errs.Add(err)
		return
	}

	v.t.Ctx.Log.Debug("Finishing poll with:\n%s", &results)
	if err := v.t.Consensus.RecordPoll(results); err != nil {
		v.t.errs.Add(err)
		return
	}

	orphans := v.t.Consensus.Orphans()
	txs := make([]snowstorm.Tx, 0, orphans.Len())
	for orphanID := range orphans {
		if tx, err := v.t.VM.GetTx(orphanID); err == nil {
			txs = append(txs, tx)
		} else {
			v.t.Ctx.Log.Warn("Failed to fetch %s during attempted re-issuance", orphanID)
		}
	}
	if len(txs) > 0 {
		v.t.Ctx.Log.Debug("Re-issuing %d transactions", len(txs))
	}
	if err := v.t.batch(txs, true /*=force*/, false /*empty*/); err != nil {
		v.t.errs.Add(err)
		return
	}

	if v.t.Consensus.Quiesce() {
		v.t.Ctx.Log.Debug("Avalanche engine can quiesce")
		return
	}

	v.t.Ctx.Log.Debug("Avalanche engine can't quiesce")
	v.t.errs.Add(v.t.repoll())
}

func (v *voter) bubbleVotes(votes ids.UniqueBag) (ids.UniqueBag, error) {
	bubbledVotes := ids.UniqueBag{}
	vertexHeap := vertex.NewHeap()
	for vote := range votes {
		vtx, err := v.t.Manager.GetVertex(vote)
		if err != nil {
			continue
		}
		vertexHeap.Push(vtx)
	}

	for vertexHeap.Len() > 0 {
		vtx := vertexHeap.Pop()
		vtxID := vtx.ID()
		set := votes.GetSet(vtxID)
		status := vtx.Status()

		if !status.Fetched() {
			v.t.Ctx.Log.Verbo("Dropping %d vote(s) for %s because the vertex is unknown",
				set.Len(), vtxID)
			bubbledVotes.RemoveSet(vtx.ID())
			continue
		}

		if status.Decided() {
			v.t.Ctx.Log.Verbo("Dropping %d vote(s) for %s because the vertex is decided",
				set.Len(), vtxID)
			bubbledVotes.RemoveSet(vtx.ID())
			continue
		}

		if v.t.Consensus.VertexIssued(vtx) {
			v.t.Ctx.Log.Verbo("Applying %d vote(s) for %s", set.Len(), vtx.ID())
			bubbledVotes.UnionSet(vtx.ID(), set)
		} else {
			v.t.Ctx.Log.Verbo("Bubbling %d vote(s) for %s because the vertex isn't issued",
				set.Len(), vtx.ID())
			bubbledVotes.RemoveSet(vtx.ID()) // Remove votes for this vertex because it hasn't been issued

			parents, err := vtx.Parents()
			if err != nil {
				return bubbledVotes, err
			}
			for _, parentVtx := range parents {
				bubbledVotes.UnionSet(parentVtx.ID(), set)
				vertexHeap.Push(parentVtx)
			}
		}
	}

	return bubbledVotes, nil
}
