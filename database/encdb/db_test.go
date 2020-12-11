// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package encdb

import (
	"testing"

	"github.com/liraxapp/avalanchego/database"
	"github.com/liraxapp/avalanchego/database/memdb"
)

func TestInterface(t *testing.T) {
	pw := "lol totally a secure password"
	for _, test := range database.Tests {
		unencryptedDB := memdb.New()
		db, err := New([]byte(pw), unencryptedDB)
		if err != nil {
			t.Fatal(err)
		}

		test(t, db)
	}
}
