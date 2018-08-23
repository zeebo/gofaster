package epoch

import (
	"sync/atomic"
)

const (
	triggerFree   = ^uint64(0)
	triggerLocked = ^uint64(0) - 1
)

type trigger struct {
	epoch  uint64
	action func()
}

// newTrigger constructs a new trigger to use.
func newTrigger() trigger { return trigger{epoch: triggerFree} }

// Epoch returns the current epoch on the trigger.
func (t *trigger) Epoch() uint64 {
	return atomic.LoadUint64(&t.epoch)
}

// Free returns true if the trigger is free for a Store.
func (t *trigger) Free() bool {
	return atomic.LoadUint64(&t.epoch) == triggerFree
}

// Run attempts to run the action stored in the trigger but only
// if the epoch matches. It returns true if the action was run.
func (t *trigger) Run(epoch uint64) bool {
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
func (t *trigger) Store(epoch uint64, action func()) bool {
	if !atomic.CompareAndSwapUint64(&t.epoch, triggerFree, triggerLocked) {
		return false
	}

	// store the action and release the lock
	t.action = action
	atomic.StoreUint64(&t.epoch, epoch)

	return true
}

// Swap attempts to swap the action stored in the trigger with the new action,
// running any old action if the epoch matches. It returns true if the swap
// was performed.
func (t *trigger) Swap(epoch, new_epoch uint64, new_action func()) bool {
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
