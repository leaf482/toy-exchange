// Package pool provides array-backed free lists.
// Get and Put do not allocate. Capacity is fixed at construction.
package pool

// FreeList is a fixed slab of T plus a stack of free indexes.
// The zero Slot value is a valid index, so callers track identity
// separately from the zero value of T.
type FreeList[T any] struct {
	slots []T
	free  []uint32
	live  []bool
	top   int32
}

// New allocates n slots. n <= 0 returns nil.
// The first Get returns slot 0, then 1, and so on, until a Put changes the order.
func New[T any](n int) *FreeList[T] {
	if n <= 0 {
		return nil
	}
	f := &FreeList[T]{
		slots: make([]T, n),
		free:  make([]uint32, n),
		live:  make([]bool, n),
		top:   int32(n),
	}
	for i := 0; i < n; i++ {
		f.free[i] = uint32(n - 1 - i)
	}
	return f
}

// Cap is the fixed number of slots.
func (f *FreeList[T]) Cap() int {
	if f == nil {
		return 0
	}
	return len(f.slots)
}

// Available is the number of slots Get can still return.
func (f *FreeList[T]) Available() int {
	if f == nil {
		return 0
	}
	return int(f.top)
}

// InUse is Cap minus Available.
func (f *FreeList[T]) InUse() int {
	if f == nil {
		return 0
	}
	return len(f.slots) - int(f.top)
}

// Get pops a free slot. The bool is false when the list is empty.
// The returned pointer addresses the slab; it is not a fresh allocation.
func (f *FreeList[T]) Get() (item *T, slot uint32, ok bool) {
	if f == nil || f.top == 0 {
		return nil, 0, false
	}
	f.top--
	slot = f.free[f.top]
	f.live[slot] = true
	return &f.slots[slot], slot, true
}

// At returns the slab pointer for slot, or nil when slot is out of range.
func (f *FreeList[T]) At(slot uint32) *T {
	if f == nil || int(slot) >= len(f.slots) {
		return nil
	}
	return &f.slots[slot]
}

// Put zeroes the slot and pushes it back onto the free stack.
// A slot that is already free, or out of range, is left unchanged.
func (f *FreeList[T]) Put(slot uint32) bool {
	if f == nil || int(slot) >= len(f.slots) || !f.live[slot] {
		return false
	}
	var zero T
	f.slots[slot] = zero
	f.live[slot] = false
	f.free[f.top] = slot
	f.top++
	return true
}
