// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package meterdb

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/liraxapp/avalanchego/database"
	"github.com/liraxapp/avalanchego/utils/timer"
)

// Database tracks the amount of time each operation takes
type Database struct {
	metrics
	db    database.Database
	clock timer.Clock
}

// New returns a new encrypted database
func New(
	namespace string,
	registerer prometheus.Registerer,
	db database.Database,
) (*Database, error) {
	meterDB := &Database{db: db}
	return meterDB, meterDB.metrics.Initialize(namespace, registerer)
}

// Has implements the Database interface
func (db *Database) Has(key []byte) (bool, error) {
	start := db.clock.Time()
	has, err := db.db.Has(key)
	end := db.clock.Time()
	db.has.Observe(float64(end.Sub(start)))
	return has, err
}

// Get implements the Database interface
func (db *Database) Get(key []byte) ([]byte, error) {
	start := db.clock.Time()
	value, err := db.db.Get(key)
	end := db.clock.Time()
	db.get.Observe(float64(end.Sub(start)))
	return value, err
}

// Put implements the Database interface
func (db *Database) Put(key, value []byte) error {
	start := db.clock.Time()
	err := db.db.Put(key, value)
	end := db.clock.Time()
	db.put.Observe(float64(end.Sub(start)))
	return err
}

// Delete implements the Database interface
func (db *Database) Delete(key []byte) error {
	start := db.clock.Time()
	err := db.db.Delete(key)
	end := db.clock.Time()
	db.delete.Observe(float64(end.Sub(start)))
	return err
}

// NewBatch implements the Database interface
func (db *Database) NewBatch() database.Batch {
	start := db.clock.Time()
	b := &batch{
		batch: db.db.NewBatch(),
		db:    db,
	}
	end := db.clock.Time()
	db.newBatch.Observe(float64(end.Sub(start)))
	return b
}

// NewIterator implements the Database interface
func (db *Database) NewIterator() database.Iterator {
	return db.NewIteratorWithStartAndPrefix(nil, nil)
}

// NewIteratorWithStart implements the Database interface
func (db *Database) NewIteratorWithStart(start []byte) database.Iterator {
	return db.NewIteratorWithStartAndPrefix(start, nil)
}

// NewIteratorWithPrefix implements the Database interface
func (db *Database) NewIteratorWithPrefix(prefix []byte) database.Iterator {
	return db.NewIteratorWithStartAndPrefix(nil, prefix)
}

// NewIteratorWithStartAndPrefix implements the Database interface
func (db *Database) NewIteratorWithStartAndPrefix(
	start,
	prefix []byte,
) database.Iterator {
	startTime := db.clock.Time()
	it := &iterator{
		iterator: db.db.NewIteratorWithStartAndPrefix(start, prefix),
		db:       db,
	}
	end := db.clock.Time()
	db.newIterator.Observe(float64(end.Sub(startTime)))
	return it
}

// Stat implements the Database interface
func (db *Database) Stat(stat string) (string, error) {
	start := db.clock.Time()
	result, err := db.db.Stat(stat)
	end := db.clock.Time()
	db.stat.Observe(float64(end.Sub(start)))
	return result, err
}

// Compact implements the Database interface
func (db *Database) Compact(start, limit []byte) error {
	startTime := db.clock.Time()
	err := db.db.Compact(start, limit)
	end := db.clock.Time()
	db.compact.Observe(float64(end.Sub(startTime)))
	return err
}

// Close implements the Database interface
func (db *Database) Close() error {
	start := db.clock.Time()
	err := db.db.Close()
	end := db.clock.Time()
	db.close.Observe(float64(end.Sub(start)))
	return err
}

type batch struct {
	batch database.Batch
	db    *Database
}

func (b *batch) Put(key, value []byte) error {
	start := b.db.clock.Time()
	err := b.batch.Put(key, value)
	end := b.db.clock.Time()
	b.db.bPut.Observe(float64(end.Sub(start)))
	return err
}

func (b *batch) Delete(key []byte) error {
	start := b.db.clock.Time()
	err := b.batch.Delete(key)
	end := b.db.clock.Time()
	b.db.bDelete.Observe(float64(end.Sub(start)))
	return err
}

func (b *batch) ValueSize() int {
	start := b.db.clock.Time()
	size := b.batch.ValueSize()
	end := b.db.clock.Time()
	b.db.bValueSize.Observe(float64(end.Sub(start)))
	return size
}

func (b *batch) Write() error {
	start := b.db.clock.Time()
	err := b.batch.Write()
	end := b.db.clock.Time()
	b.db.bWrite.Observe(float64(end.Sub(start)))
	return err
}

func (b *batch) Reset() {
	start := b.db.clock.Time()
	b.batch.Reset()
	end := b.db.clock.Time()
	b.db.bReset.Observe(float64(end.Sub(start)))
}

func (b *batch) Replay(w database.KeyValueWriter) error {
	start := b.db.clock.Time()
	err := b.batch.Replay(w)
	end := b.db.clock.Time()
	b.db.bReplay.Observe(float64(end.Sub(start)))
	return err
}

func (b *batch) Inner() database.Batch {
	start := b.db.clock.Time()
	inner := b.batch.Inner()
	end := b.db.clock.Time()
	b.db.bInner.Observe(float64(end.Sub(start)))
	return inner
}

type iterator struct {
	iterator database.Iterator
	db       *Database
}

func (it *iterator) Next() bool {
	start := it.db.clock.Time()
	next := it.iterator.Next()
	end := it.db.clock.Time()
	it.db.iNext.Observe(float64(end.Sub(start)))
	return next
}

func (it *iterator) Error() error {
	start := it.db.clock.Time()
	err := it.iterator.Error()
	end := it.db.clock.Time()
	it.db.iError.Observe(float64(end.Sub(start)))
	return err
}

func (it *iterator) Key() []byte {
	start := it.db.clock.Time()
	key := it.iterator.Key()
	end := it.db.clock.Time()
	it.db.iKey.Observe(float64(end.Sub(start)))
	return key
}

func (it *iterator) Value() []byte {
	start := it.db.clock.Time()
	value := it.iterator.Value()
	end := it.db.clock.Time()
	it.db.iValue.Observe(float64(end.Sub(start)))
	return value
}

func (it *iterator) Release() {
	start := it.db.clock.Time()
	it.iterator.Release()
	end := it.db.clock.Time()
	it.db.iRelease.Observe(float64(end.Sub(start)))
}
