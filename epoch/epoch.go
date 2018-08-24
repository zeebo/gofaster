package epoch

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/zeebo/gofaster/internal/machine"
)

// TODO(jeff): we use atomic loads everywhere when they might not be totally
// necessary. can they be relaxed, and will the race detector complain?

const (
	epochMaxTriggers = 256
)

var epochData struct {
	// keep track of the current epoch
	current uint64
	_       [56]uint8

	// keep track of which epoch is safe
	safe uint64
	_    [56]uint8

	// keep track of entries
	entries [machine.MaxThreads]entry

	// keep track of triggers
	trigger_count uint64
	_             [56]uint8
	triggers      [epochMaxTriggers]trigger
}

func init() {
	epochData.current = 1
	for i := range &epochData.triggers {
		epochData.triggers[i] = newTrigger()
	}
}

// getEntry returns the entry bound to the given handle id.
func getEntry(id uint32) *entry {
	return &epochData.entries[id%machine.MaxThreads]
}

// Protect enters the protected region of the epoch.
func Protect(h Handle) uint64 {
	entry := getEntry(h.Id())
	entry.local = atomic.LoadUint64(&epochData.current)
	return entry.local
}

// ProtectAndDrain enters the protected region of the epoch, draining any triggers if possible.
func ProtectAndDrain(h Handle) uint64 {
	epoch := Protect(h)
	if atomic.LoadUint64(&epochData.trigger_count) > 0 {
		Drain(epoch)
	}
	return epoch
}

// IsProtected returns if the handle is in the protected region.
func IsProtected(h Handle) bool {
	entry := getEntry(h.Id())
	return entry.local != 0
}

// Unprotect exits the protected region.
func Unprotect(h Handle) {
	entry := getEntry(h.Id())
	entry.local = 0
}

// Drain runs any triggers that are safe to run. The provided epoch is used as an
// initial epoch for computing which epoch is safe.
func Drain(epoch uint64) {
	ComputeSafe(epoch)

	for i := range &epochData.triggers {
		trigger := &epochData.triggers[i]
		epoch := trigger.Epoch()
		safe := atomic.LoadUint64(&epochData.safe)

		if epoch <= safe &&
			trigger.Run(epoch) &&
			atomic.AddUint64(&epochData.trigger_count, ^uint64(0)) == 0 {

			break
		}
	}
}

// Bump increments the global epoch, draining any triggers that can be drained.
func Bump() uint64 {
	epoch := atomic.AddUint64(&epochData.current, 1)
	if atomic.LoadUint64(&epochData.trigger_count) > 0 {
		Drain(epoch)
	}
	return epoch
}

// BumpWith increments the global epoch and adds the action into the trigger queue.
func BumpWith(action func()) uint64 {
	prior := Bump() - 1
	failures := 0

finished:
	for {
		for i := range &epochData.triggers {
			trigger := &epochData.triggers[i]
			epoch := trigger.Epoch()

			if epoch == triggerFree && trigger.Store(epoch, action) {
				break finished
			}

			safe := atomic.LoadUint64(&epochData.safe)
			if epoch <= safe && trigger.Swap(epoch, prior, action) {
				break finished
			}
		}

		failures++
		if failures == 500 {
			failures = 0
			fmt.Fprintln(os.Stderr, "Slowdown: Unable to add trigger to epoch")
			time.Sleep(time.Second)
		}
	}

	atomic.AddUint64(&epochData.trigger_count, 1)
	return prior + 1
}

// ComputeSafe finds the current safe epoch across all the entries, using the
// provided epoch as an initial value.
func ComputeSafe(epoch uint64) uint64 {
	oldest := epoch
	for i := range &epochData.entries {
		local := atomic.LoadUint64(&epochData.entries[i].local)
		if local != 0 && local < oldest {
			oldest = local
		}
	}
	atomic.StoreUint64(&epochData.safe, oldest-1)
	return oldest - 1
}
