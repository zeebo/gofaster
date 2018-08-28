package htable

import (
	"bytes"
	"unsafe"

	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/machine"
	"github.com/zeebo/gofaster/pin"
)

// bucket is a cache line sized array of entries for a hash table, with the
// last entry being a pointer to an overflow bucket.
type bucket struct {
	entries  [7]pin.Location
	overflow *bucket
}

const bucketSize = unsafe.Sizeof(bucket{})

type ( // ensure the bucket is sized to a cache line
	_ [bucketSize - machine.CacheLine]byte
	_ [machine.CacheLine - bucketSize]byte
)

// Delete removes the key from the bucket, using the tag to avoid comparing keys.
// It returns false if the key does not exist.
func (b *bucket) Delete(h epoch.Handle, ex uint16, key []byte) (bool, bool) {
	for i := range &b.entries {
		addr := &b.entries[i]
		loc := pin.LoadLocation(addr)
		t := tag(loc.Extra())

		// check if the tag is appropriate
		if loc.Nil() || t.Hash() != ex || t.Tentative() {
			continue
		}

		// the key must exist in this bucket entry if it exists
	retry:
		caddr := addr

		for {
			cloc := pin.LoadLocation(caddr)
			if cloc.Nil() {
				break
			}

			rec := (*record)(pin.Read(cloc))
			if !bytes.Equal(rec.Key(), key) {
				caddr = &rec.next
				continue
			}

			// grab the original next pointer
			rloc := pin.LoadLocation(&rec.next)
			rtag := tag(rloc.Extra())

			{ // flag the pointer on the record as logically deleted
				nloc := rloc.WithExtra(uint16(rtag.WithDelete()))
				if !pin.CompareAndSwapLocation(&rec.next, rloc, nloc) {
					goto retry
				}
			}

			{ // clear out any possible delete flag and go from cloc => rec.next
				nloc := rloc.WithExtra(uint16(rtag.WithoutDelete()))
				if !pin.CompareAndSwapLocation(caddr, cloc, nloc) {
					pin.StoreLocation(&rec.next, rloc)
					goto retry
				}
			}

			// we use the epoch system to unpin the deleted location which ensures
			// no other handles are reading.
			epoch.BumpWith(h, func(h epoch.Handle) { pin.Unpin(h, cloc) })

			return true, true
		}

		// we failed to delete
		return true, false
	}

	return false, false
}

// Lookup returns the value for the key, using the tag to avoid comparing keys.
// It returns nil if the key does not exist.
func (b *bucket) Lookup(h epoch.Handle, ex uint16, key []byte) (bool, []byte) {
	for i := range &b.entries {
		addr := &b.entries[i]
		loc := pin.LoadLocation(addr)
		t := tag(loc.Extra())

		// check if the tag is appropriate
		if loc.Nil() || t.Hash() != ex || t.Tentative() {
			continue
		}

		// check the linked list of records for the matching key
		for !loc.Nil() {
			rec := (*record)(pin.Read(loc))
			if bytes.Equal(rec.Key(), key) {
				return true, rec.Val()
			}
			loc = pin.LoadLocation(&rec.next)
		}

		return true, nil
	}

	return false, nil
}

// Insert adds the location to the bucket using the extra hash to find the correct index location.
func (b *bucket) Insert(h epoch.Handle, loc pin.Location, key []byte) bool {
	ex := tag(loc.Extra()).Hash()

retry:
	for i := range &b.entries {
		caddr := &b.entries[i]
		cloc := pin.LoadLocation(caddr)
		ctag := tag(cloc.Extra())

		if cloc.Nil() || ctag.Hash() != ex || ctag.Tentative() {
			continue
		}

		// walk the records to see if we already have the key
		for !cloc.Nil() {
			rec := (*record)(pin.Read(cloc))
			if bytes.Equal(rec.Key(), key) {
				// TODO(jeff): update this record to be the right one? this is weird
				return true
			}
			cloc = pin.LoadLocation(&rec.next)
		}

		// read the record, and update next to point at the loaded location
		rec := (*record)(pin.Read(loc))
		pin.StoreLocation(&rec.next, cloc)

		// attempt to append our record to the start of the linked list. if we
		// fail, retry everything.
		if !pin.CompareAndSwapLocation(caddr, cloc, loc) {
			goto retry
		}

		// the insert is complete
		return true
	}

	return false
}
