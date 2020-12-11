// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cache

import (
	"github.com/liraxapp/avalanchego/ids"
)

// Cacher acts as a best effort key value store
type Cacher interface {
	// Put inserts an element into the cache. If spaced is required, elements will
	// be evicted.
	Put(key ids.ID, value interface{})

	// Get returns the entry in the cache with the key specified, if no value
	// exists, false is returned.
	Get(key ids.ID) (interface{}, bool)

	// Evict removes the specified entry from the cache
	Evict(key ids.ID)

	// Flush removes all entries from the cache
	Flush()
}

// Evictable allows the object to be notified when it is evicted
type Evictable interface {
	ID() ids.ID
	Evict()
}

// Deduplicator acts as a best effort deduplication service
type Deduplicator interface {
	// Deduplicate returns either the provided value, or a previously provided
	// value with the same ID that hasn't yet been evicted
	Deduplicate(Evictable) Evictable

	// Flush removes all entries from the cache
	Flush()
}
