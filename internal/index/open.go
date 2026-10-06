package index

const (
	stateEmpty uint8 = 0
	stateLive  uint8 = 1
	stateTomb  uint8 = 2
)

// Open is a fixed open-addressing map from order id to pool slot.
// Insert does not grow the table. Past a 0.7 load it returns false.
type Open struct {
	keys  []uint64
	slots []uint32
	state []uint8
	used  int32
	mask  uint32
}

// NewOpen sizes a power-of-two table that can hold n live keys under a 0.7 load.
// n <= 0 returns nil.
func NewOpen(n int) *Open {
	if n <= 0 {
		return nil
	}
	need := n*10/7 + 1
	size := 2
	for size < need {
		size <<= 1
	}
	return &Open{
		keys:  make([]uint64, size),
		slots: make([]uint32, size),
		state: make([]uint8, size),
		mask:  uint32(size - 1),
	}
}

func mix(id uint64) uint64 {
	z := id * 11400714819323198485
	z ^= z >> 32
	return z
}

// Lookup returns the pool slot for id.
func (o *Open) Lookup(id uint64) (uint32, bool) {
	if o == nil || id == 0 {
		return 0, false
	}
	i := uint32(mix(id)) & o.mask
	start := i
	for {
		switch o.state[i] {
		case stateEmpty:
			return 0, false
		case stateLive:
			if o.keys[i] == id {
				return o.slots[i], true
			}
		}
		i = (i + 1) & o.mask
		if i == start {
			return 0, false
		}
	}
}

// Has reports whether id is present.
func (o *Open) Has(id uint64) bool {
	_, ok := o.Lookup(id)
	return ok
}

// Insert stores id unless it is 0, already present, or the table would pass a 0.7 load.
func (o *Open) Insert(id uint64, slot uint32) bool {
	if o == nil || id == 0 {
		return false
	}
	if int(o.used+1)*10 > len(o.keys)*7 {
		return false
	}
	i := uint32(mix(id)) & o.mask
	start := i
	tomb := -1
	for {
		switch o.state[i] {
		case stateEmpty:
			if tomb >= 0 {
				i = uint32(tomb)
			}
			o.state[i] = stateLive
			o.keys[i] = id
			o.slots[i] = slot
			o.used++
			return true
		case stateTomb:
			if tomb < 0 {
				tomb = int(i)
			}
		default:
			if o.keys[i] == id {
				return false
			}
		}
		i = (i + 1) & o.mask
		if i == start {
			if tomb >= 0 {
				i = uint32(tomb)
				o.state[i] = stateLive
				o.keys[i] = id
				o.slots[i] = slot
				o.used++
				return true
			}
			return false
		}
	}
}

// Delete removes id and leaves a tombstone so probe chains stay intact.
func (o *Open) Delete(id uint64) {
	if o == nil || id == 0 {
		return
	}
	i := uint32(mix(id)) & o.mask
	start := i
	for {
		switch o.state[i] {
		case stateEmpty:
			return
		case stateLive:
			if o.keys[i] == id {
				o.state[i] = stateTomb
				o.keys[i] = 0
				o.slots[i] = 0
				o.used--
				return
			}
		}
		i = (i + 1) & o.mask
		if i == start {
			return
		}
	}
}

// Len is the number of live ids.
func (o *Open) Len() int {
	if o == nil {
		return 0
	}
	return int(o.used)
}

// Range visits every live id.
func (o *Open) Range(fn func(id uint64, slot uint32)) {
	if o == nil {
		return
	}
	for i, st := range o.state {
		if st == stateLive {
			fn(o.keys[i], o.slots[i])
		}
	}
}
