package htable

import (
	"sync/atomic"
	"unsafe"

	"github.com/cespare/xxhash"
	"github.com/zeebo/gofaster/epoch"
	"github.com/zeebo/gofaster/internal/risky"
	"github.com/zeebo/gofaster/pin"
)

// Table is a concurrent hash table.
type Table struct {
	buckets []bucket
	bits    uint64 // 2^bits buckets
	mask    uint64
	ops     uint64
}

// New constructs a table with 2^bits buckets.
func New(bits uint64) *Table {
	return &Table{
		buckets: make([]bucket, 1<<bits),
		bits:    bits,
		mask:    1<<bits - 1,
	}
}

// split turns the hash into ex hash bits and bucket index.
func (t *Table) split(hash uint64) (uint16, uint64) {
	return uint16(hash) & tagHashMask, hash >> tagHashBits & t.mask
}

// index returns the bucket for the given index.
func (t *Table) index(i uint64) *bucket {
	ptr := risky.Index(unsafe.Pointer(&t.buckets), bucketSize, uintptr(i))
	return (*bucket)(unsafe.Pointer(ptr))
}

// protect enters a protected region for the handle, draining the epoch queue periodically.
func (t *Table) protect(h epoch.Handle) {
	if atomic.AddUint64(&t.ops, 1)%512 == 0 {
		epoch.ProtectAndDrain(h)
	} else {
		epoch.Protect(h)
	}
}

// Delete removes the key from the table and returns true if it was able to.
func (t *Table) Delete(h epoch.Handle, key []byte) bool {
	t.protect(h)

	ex, idx := t.split(xxhash.Sum64(key))
	for bucket := t.index(idx); bucket != nil; bucket = bucket.overflow {
		if found, deleted := bucket.Delete(h, ex, key); found {
			epoch.Unprotect(h)
			return deleted
		}
	}

	epoch.Unprotect(h)
	return false
}

// Lookup finds the value for the key, returning nil if no key matches.
func (t *Table) Lookup(h epoch.Handle, key []byte) []byte {
	t.protect(h)

	ex, idx := t.split(xxhash.Sum64(key))
	for bucket := t.index(idx); bucket != nil; bucket = bucket.overflow {
		if found, val := bucket.Lookup(h, ex, key); found {
			epoch.Unprotect(h)
			return val
		}
	}

	epoch.Unprotect(h)
	return nil
}

// Insert adds the key and value to the table.
func (t *Table) Insert(h epoch.Handle, key, value []byte) {
	t.protect(h)

	rec := newRecord(key, value)
	ex, idx := t.split(xxhash.Sum64(key))
	loc := pin.Pin(h, unsafe.Pointer(rec)).WithExtra(ex)
	tloc := loc.WithExtra(uint16(tag(ex).WithTentative()))

retry:

	// first attempt to find a bucket with a matching tag already
	for bucket := t.index(idx); bucket != nil; bucket = bucket.overflow {
		if bucket.Insert(h, loc, key) {
			epoch.Unprotect(h)
			return
		}
	}

	// we didn't find one. find an empty spot or allocate an overflow bucket
	var last *bucket
	for bucket := t.index(idx); bucket != nil; bucket = bucket.overflow {
		last = bucket

		for i := range &bucket.entries {
			caddr := &bucket.entries[i]
			cloc := pin.LoadLocation(caddr)

			// if we find a nil location, attempt to add our loc in
			if cloc.Nil() {
				if !pin.CompareAndSwapLocation(caddr, cloc, tloc) {
					goto retry
				}

				// now we have to rescan the buckets for any matching tentative locations
				match := false
			searching:
				for bucket := t.index(idx); bucket != nil; bucket = bucket.overflow {
					for i := range &bucket.entries {
						maddr := &bucket.entries[i]
						if caddr == maddr {
							continue
						}
						mex := tag(pin.LoadLocation(maddr).Extra()).Hash()
						if mex == ex {
							match = true
							break searching
						}
					}
				}

				// if there was a match, abort and retry
				if match {
					pin.StoreLocation(caddr, pin.Location{})
					goto retry
				}

				// otherwise, we won with no contention, so clear tentative bit
				pin.StoreLocation(caddr, loc)
				epoch.Unprotect(h)
				return
			}
		}
	}

	// we found no empty spot. allocate an overflow bucket and retry
	ptr := (*unsafe.Pointer)(unsafe.Pointer(&last.overflow))
	atomic.CompareAndSwapPointer(ptr, nil, unsafe.Pointer(new(bucket)))
	goto retry
}
