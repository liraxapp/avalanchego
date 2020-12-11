// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ids

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/liraxapp/avalanchego/utils/formatting"
	"github.com/liraxapp/avalanchego/utils/hashing"
)

// ShortEmpty is a useful all zero value
var ShortEmpty = ShortID{ID: &[20]byte{}}

// ShortID wraps a 20 byte hash as an identifier
type ShortID struct {
	ID *[20]byte `serialize:"true"`
}

// NewShortID creates an identifier from a 20 byte hash
func NewShortID(id [20]byte) ShortID { return ShortID{ID: &id} }

// ToShortID attempt to convert a byte slice into an id
func ToShortID(bytes []byte) (ShortID, error) {
	addrHash, err := hashing.ToHash160(bytes)
	return NewShortID(addrHash), err
}

// ShortFromString is the inverse of ShortID.String()
func ShortFromString(idStr string) (ShortID, error) {
	bytes, err := formatting.Decode(defaultEncoding, idStr)
	if err != nil {
		return ShortID{}, err
	}
	return ToShortID(bytes)
}

// ShortFromPrefixedString returns a ShortID assuming the cb58 format is
// prefixed
func ShortFromPrefixedString(idStr, prefix string) (ShortID, error) {
	if !strings.HasPrefix(idStr, prefix) {
		return ShortID{}, fmt.Errorf("ID: %s is missing the prefix: %s", idStr, prefix)
	}

	return ShortFromString(strings.TrimPrefix(idStr, prefix))
}

// MarshalJSON ...
func (id ShortID) MarshalJSON() ([]byte, error) {
	if id.IsZero() {
		return []byte("null"), nil
	}
	str, err := formatting.Encode(defaultEncoding, id.ID[:])
	if err != nil {
		return nil, err
	}
	return []byte("\"" + str + "\""), nil
}

// UnmarshalJSON ...
func (id *ShortID) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == "null" { // If "null", do nothing
		return nil
	} else if len(str) < 2 {
		return errMissingQuotes
	}

	lastIndex := len(str) - 1
	if str[0] != '"' || str[lastIndex] != '"' {
		return errMissingQuotes
	}

	// Parse CB58 formatted string to bytes
	bytes, err := formatting.Decode(defaultEncoding, str[1:lastIndex])
	if err != nil {
		return fmt.Errorf("couldn't decode ID to bytes: %w", err)
	}
	*id, err = ToShortID(bytes)
	return err
}

// IsZero returns true if the value has not been initialized
func (id ShortID) IsZero() bool { return id.ID == nil }

// Key returns a 20 byte hash that this id represents. This is useful to allow
// for this id to be used as keys in maps.
func (id ShortID) Key() [20]byte { return *id.ID }

// Equals returns true if the ids have the same byte representation
func (id ShortID) Equals(oID ShortID) bool {
	return id.ID == oID.ID ||
		(id.ID != nil && oID.ID != nil && bytes.Equal(id.Bytes(), oID.Bytes()))
}

// Bytes returns the 20 byte hash as a slice. It is assumed this slice is not
// modified.
func (id ShortID) Bytes() []byte { return id.ID[:] }

// Hex returns a hex encoded string of this id.
func (id ShortID) Hex() string { return hex.EncodeToString(id.Bytes()) }

func (id ShortID) String() string {
	if id.IsZero() {
		return "nil"
	}
	// We assume that the maximum size of a byte slice that
	// can be stringified is at least the length of an ID
	str, _ := formatting.Encode(defaultEncoding, id.Bytes())
	return str
}

// PrefixedString returns the String representation with a prefix added
func (id ShortID) PrefixedString(prefix string) string {
	return prefix + id.String()
}

type sortShortIDData []ShortID

func (ids sortShortIDData) Less(i, j int) bool {
	return bytes.Compare(
		ids[i].Bytes(),
		ids[j].Bytes()) == -1
}
func (ids sortShortIDData) Len() int      { return len(ids) }
func (ids sortShortIDData) Swap(i, j int) { ids[j], ids[i] = ids[i], ids[j] }

// SortShortIDs sorts the ids lexicographically
func SortShortIDs(ids []ShortID) { sort.Sort(sortShortIDData(ids)) }

// IsSortedAndUniqueShortIDs returns true if the ids are sorted and unique
func IsSortedAndUniqueShortIDs(ids []ShortID) bool {
	for i := 0; i < len(ids)-1; i++ {
		if bytes.Compare(ids[i].Bytes(), ids[i+1].Bytes()) != -1 {
			return false
		}
	}
	return true
}

// IsUniqueShortIDs returns true iff [ids] are unique
func IsUniqueShortIDs(ids []ShortID) bool {
	set := ShortSet{}
	set.Add(ids...)
	return set.Len() == len(ids)
}
