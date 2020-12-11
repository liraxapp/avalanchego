// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gsharedmemory

import (
	"context"

	"github.com/liraxapp/avalanchego/chains/atomic"
	"github.com/liraxapp/avalanchego/database"
	"github.com/liraxapp/avalanchego/ids"
	"github.com/liraxapp/avalanchego/vms/rpcchainvm/gsharedmemory/gsharedmemoryproto"
)

// Server is a messenger that is managed over RPC.
type Server struct {
	sm atomic.SharedMemory
	db database.Database
}

// NewServer returns a vm instance connected to a remote vm instance
func NewServer(sm atomic.SharedMemory, db database.Database) *Server {
	return &Server{
		sm: sm,
		db: db,
	}
}

// Put ...
func (s *Server) Put(
	_ context.Context,
	req *gsharedmemoryproto.PutRequest,
) (*gsharedmemoryproto.PutResponse, error) {
	peerChainID, err := ids.ToID(req.PeerChainID)
	if err != nil {
		return nil, err
	}

	elems := make([]*atomic.Element, len(req.Elems))
	for i, elem := range req.Elems {
		elems[i] = &atomic.Element{
			Key:    elem.Key,
			Value:  elem.Value,
			Traits: elem.Traits,
		}
	}

	batches := make([]database.Batch, len(req.Batches))
	for i, reqBatch := range req.Batches {
		batch := s.db.NewBatch()
		for _, putReq := range reqBatch.Puts {
			if err := batch.Put(putReq.Key, putReq.Value); err != nil {
				return nil, err
			}
		}

		for _, deleteReq := range reqBatch.Deletes {
			if err := batch.Delete(deleteReq.Key); err != nil {
				return nil, err
			}
		}
		batches[i] = batch
	}

	return &gsharedmemoryproto.PutResponse{}, s.sm.Put(peerChainID, elems, batches...)
}

// Get ...
func (s *Server) Get(
	_ context.Context,
	req *gsharedmemoryproto.GetRequest,
) (*gsharedmemoryproto.GetResponse, error) {
	peerChainID, err := ids.ToID(req.PeerChainID)
	if err != nil {
		return nil, err
	}

	values, err := s.sm.Get(peerChainID, req.Keys)
	return &gsharedmemoryproto.GetResponse{
		Values: values,
	}, err
}

// Indexed ...
func (s *Server) Indexed(
	_ context.Context,
	req *gsharedmemoryproto.IndexedRequest,
) (*gsharedmemoryproto.IndexedResponse, error) {
	peerChainID, err := ids.ToID(req.PeerChainID)
	if err != nil {
		return nil, err
	}

	values, lastTrait, lastKey, err := s.sm.Indexed(
		peerChainID,
		req.Traits,
		req.StartTrait,
		req.StartKey,
		int(req.Limit),
	)
	return &gsharedmemoryproto.IndexedResponse{
		Values:    values,
		LastTrait: lastTrait,
		LastKey:   lastKey,
	}, err
}

// Remove ...
func (s *Server) Remove(
	_ context.Context,
	req *gsharedmemoryproto.RemoveRequest,
) (*gsharedmemoryproto.RemoveResponse, error) {
	peerChainID, err := ids.ToID(req.PeerChainID)
	if err != nil {
		return nil, err
	}

	batches := make([]database.Batch, len(req.Batches))
	for i, reqBatch := range req.Batches {
		batch := s.db.NewBatch()
		for _, putReq := range reqBatch.Puts {
			if err := batch.Put(putReq.Key, putReq.Value); err != nil {
				return nil, err
			}
		}

		for _, deleteReq := range reqBatch.Deletes {
			if err := batch.Delete(deleteReq.Key); err != nil {
				return nil, err
			}
		}
		batches[i] = batch
	}

	return &gsharedmemoryproto.RemoveResponse{}, s.sm.Remove(peerChainID, req.Keys, batches...)
}
