// Package index maps an order id to a pool slot.
// Phase 2 uses a builtin map. Phase 4 replaces the book with an open-addressing table.
package index

// Map is the baseline id index. Insert may allocate.
type Map struct {
	m map[uint64]uint32
}

// NewMap allocates a map with room for hint entries.
// hint <= 0 returns nil.
func NewMap(hint int) *Map {
	if hint <= 0 {
		return nil
	}
	return &Map{m: make(map[uint64]uint32, hint)}
}

// Lookup returns the pool slot for id.
func (m *Map) Lookup(id uint64) (uint32, bool) {
	if m == nil {
		return 0, false
	}
	slot, ok := m.m[id]
	return slot, ok
}

// Has reports whether id is present.
func (m *Map) Has(id uint64) bool {
	_, ok := m.Lookup(id)
	return ok
}

// Insert stores id unless it is 0 or already present.
func (m *Map) Insert(id uint64, slot uint32) bool {
	if m == nil || id == 0 {
		return false
	}
	if _, ok := m.m[id]; ok {
		return false
	}
	m.m[id] = slot
	return true
}

// Delete removes id. A missing id is a no-op.
func (m *Map) Delete(id uint64) {
	if m == nil {
		return
	}
	delete(m.m, id)
}

// Len is the number of live ids.
func (m *Map) Len() int {
	if m == nil {
		return 0
	}
	return len(m.m)
}

// Range visits every live id. Order is not defined.
func (m *Map) Range(fn func(id uint64, slot uint32)) {
	if m == nil {
		return
	}
	for id, slot := range m.m {
		fn(id, slot)
	}
}
