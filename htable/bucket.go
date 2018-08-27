package htable

import (
	"bytes"
	"sync/atomic"
	"unsafe"

	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/machine"
	"github.com/zeebo/gofaster/pin"
)

const (
	tentativeBit = 1 << 15
)

// bucket is a cache line sized array of entries for a hash table, with the
// last entry being a pointer to an overflow bucket.
type bucket struct {
	entries  [7]uint64
	overflow *bucket
}

const bucketSize = unsafe.Sizeof(bucket{})

type ( // ensure the bucket is sized to a cache line
	_ [bucketSize - machine.CacheLine]byte
	_ [machine.CacheLine - bucketSize]byte
)

// Delete removes the key from the bucket, using the tag to avoid comparing keys.
// It returns false if the key does not exist.
func (b *bucket) Delete(h epoch.Handle, tag uint16, key []byte) (bool, bool) {
	for i := range &b.entries {
		addr := &b.entries[i]
		val := atomic.LoadUint64(addr)
		loc := pin.Location(val)

		// first check the tag/tentative bit to see if this is the right entry
		if extra := loc.Extra(); extra&tagMask != tag && extra&tentativeBit > 0 {
			continue
		}

		// then do an expensive key comparison
		for loc != 0 {
			rec := (*record)(pin.Read(loc))

			if !bytes.Equal(rec.Key(), key) {
				loc = rec.next
				continue
			}

		}
		// attempt the delete
		return true, atomic.CompareAndSwapUint64(addr, val, 0)
	}

	return false, false
}

// Lookup returns the value for the key, using the tag to avoid comparing keys.
// It returns nil if the key does not exist.
func (b *bucket) Lookup(h epoch.Handle, tag uint16, key []byte) (bool, []byte) {
	for i := range &b.entries {
		addr := &b.entries[i]
		val := atomic.LoadUint64(addr)
		loc := pin.Location(val)

		// first check the tag/tentative bit to see if this is the right entry
		if extra := loc.Extra(); extra&tagMask != tag && extra&tentativeBit > 0 {
			continue
		}

		// check the linked list of records for the matching key
		for loc != 0 {
			rec := (*record)(pin.Read(loc))
			if bytes.Equal(rec.Key(), key) {
				return true, rec.Val()
			}
			loc = rec.next
		}

		return true, nil
	}

	return false, nil
}
