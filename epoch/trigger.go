package epoch

import (
	"sync/atomic"
)

const (
	triggerFree   = ^uint64(0)
	triggerLocked = ^uint64(0) - 1
)

type Trigger struct {
	epoch  uint64
	action func()
}

// Epoch returns the current epoch on the Trigger.
func (t *Trigger) Epoch() uint64 {
	return atomic.LoadUint64(&t.epoch)
}

// Free returns true if the trigger is free for a Store.
func (t *Trigger) Free() bool {
	return atomic.LoadUint64(&t.epoch) == triggerFree
}

// Run attempts to run the action stored in the Trigger but only
// if the epoch matches. It returns true if the action was run.
func (t *Trigger) Run(epoch uint64) bool {
	if !atomic.CompareAndSwapUint64(&t.epoch, epoch, triggerLocked) {
		return false
	}

	// acquire the action and release the lock
	action := t.action
	t.action = nil
	atomic.StoreUint64(&t.epoch, triggerFree)

	action()

	return true
}

// Store attempts to store the action to be run after the given
// epoch, and returns true if it was able to store it.
func (t *Trigger) Store(epoch uint64, action func()) bool {
	if !atomic.CompareAndSwapUint64(&t.epoch, triggerFree, triggerLocked) {
		return false
	}

	// store the action and release the lock
	t.action = action
	atomic.StoreUint64(&t.epoch, epoch)

	return true
}

// Swap attempts to swap the action stored in the Trigger with the new action,
// running any old action if the epoch matches. It returns true if the swap
// was performed.
func (t *Trigger) Swap(epoch, new_epoch uint64, new_action func()) bool {
	if !atomic.CompareAndSwapUint64(&t.epoch, epoch, new_epoch) {
		return false
	}

	// acquire the action, store the new action, and release the lock
	action := t.action
	t.action = new_action
	atomic.StoreUint64(&t.epoch, new_epoch)

	action()

	return true
}
