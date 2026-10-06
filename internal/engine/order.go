package engine

import "github.com/leaf482/toy-exchange/internal/pool"

// Order is one order. Prev and Next are the FIFO links at its price.
// A resting order occupies one pool slot and one level.
type Order struct {
	ID        uint64
	Side      Side
	Type      OrderType
	Price     int64
	Qty       int64
	Leaves    int64
	Timestamp int64
	Prev      *Order
	Next      *Order
	Level     *LimitLevel
	Slot      uint32
}

// Side is the order side.
type Side uint8

const (
	SideBuy  Side = 1
	SideSell Side = 2
)

// OrderType is a limit or a market order.
type OrderType uint8

const (
	TypeLimit  OrderType = 1
	TypeMarket OrderType = 2
)

// LimitLevel is one price. Orders are a doubly linked FIFO queue.
// Left, Right, Parent, and Color belong to the price tree.
type LimitLevel struct {
	Price    int64
	TotalQty int64
	Head     *Order
	Tail     *Order
	Count    int32
	Left     *LimitLevel
	Right    *LimitLevel
	Parent   *LimitLevel
	Color    uint8 // 0 = black, 1 = red
	Slot     uint32
}

// OrderPool is a fixed slab of Order values.
type OrderPool struct {
	free *pool.FreeList[Order]
}

// NewOrderPool allocates n order slots. n <= 0 returns nil.
func NewOrderPool(n int) *OrderPool {
	f := pool.New[Order](n)
	if f == nil {
		return nil
	}
	return &OrderPool{free: f}
}

// Get returns a zeroed order from the slab and records its slot.
func (p *OrderPool) Get() (*Order, bool) {
	if p == nil {
		return nil, false
	}
	o, slot, ok := p.free.Get()
	if !ok {
		return nil, false
	}
	o.Slot = slot
	return o, true
}

// Put returns an order to the slab. The slot is zeroed.
// An order that did not come from this pool is ignored.
func (p *OrderPool) Put(o *Order) {
	if p == nil || o == nil {
		return
	}
	slot := o.Slot
	if p.free.At(slot) != o {
		return
	}
	p.free.Put(slot)
}

// Cap is the fixed number of order slots.
func (p *OrderPool) Cap() int {
	if p == nil {
		return 0
	}
	return p.free.Cap()
}

// At returns the slab pointer for a slot.
func (p *OrderPool) At(slot uint32) *Order {
	if p == nil {
		return nil
	}
	return p.free.At(slot)
}

// InUse is the number of orders currently taken from the pool.
func (p *OrderPool) InUse() int {
	if p == nil {
		return 0
	}
	return p.free.InUse()
}

// LevelPool is a fixed slab of LimitLevel values.
type LevelPool struct {
	free *pool.FreeList[LimitLevel]
}

// NewLevelPool allocates n level slots. n <= 0 returns nil.
func NewLevelPool(n int) *LevelPool {
	f := pool.New[LimitLevel](n)
	if f == nil {
		return nil
	}
	return &LevelPool{free: f}
}

// Get returns a zeroed level from the slab and records its slot.
func (p *LevelPool) Get() (*LimitLevel, bool) {
	if p == nil {
		return nil, false
	}
	lvl, slot, ok := p.free.Get()
	if !ok {
		return nil, false
	}
	lvl.Slot = slot
	return lvl, true
}

// Put returns a level to the slab. The slot is zeroed.
// A level that did not come from this pool is ignored.
func (p *LevelPool) Put(lvl *LimitLevel) {
	if p == nil || lvl == nil {
		return
	}
	slot := lvl.Slot
	if p.free.At(slot) != lvl {
		return
	}
	p.free.Put(slot)
}
