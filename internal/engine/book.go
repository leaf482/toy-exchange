package engine

import (
	"errors"

	"github.com/leaf482/toy-exchange/internal/index"
	"github.com/leaf482/toy-exchange/pkg/events"
)

// ErrBufferFull means the caller buffer cannot hold the worst-case events.
// The book is unchanged.
var ErrBufferFull = errors.New("event buffer full")

// ErrNotFound means the id is not a resting order.
var ErrNotFound = errors.New("order not found")

// ErrBadAmend means the amend price or quantity breaks the amend rules.
var ErrBadAmend = errors.New("bad amend")

// ErrCapacity means a pool or index capacity is not positive.
var ErrCapacity = errors.New("invalid capacity")

// OrderInput is the caller-owned description of a new order.
// The engine copies it into a pool slot only when the order rests.
type OrderInput struct {
	ID        uint64
	Side      Side
	Type      OrderType
	Price     int64
	Qty       int64
	Timestamp int64
}

// LevelView is one price level in a depth snapshot.
type LevelView struct {
	Price int64
	Qty   int64
	Count int32
}

// OrderBook is the single-threaded price-time book.
// bestBid and bestAsk are the spec's cached best levels. The names differ
// from the BestBid and BestAsk methods, which return price and size.
type OrderBook struct {
	Bids         PriceTree
	Asks         PriceTree
	bestBid      *LimitLevel
	bestAsk      *LimitLevel
	orders       *index.Map
	orderPool    *OrderPool
	levelPool    *LevelPool
	NextTradeID  uint64
	RestingCount int32
}

// New builds a book with fixed pool capacities.
// indexCap sizes the baseline id map.
func New(orderCap, levelCap, indexCap int) (*OrderBook, error) {
	if orderCap <= 0 || levelCap <= 0 || indexCap <= 0 {
		return nil, ErrCapacity
	}
	orders := index.NewMap(indexCap)
	op := NewOrderPool(orderCap)
	lp := NewLevelPool(levelCap)
	if orders == nil || op == nil || lp == nil {
		return nil, ErrCapacity
	}
	return &OrderBook{
		orders:      orders,
		orderPool:   op,
		levelPool:   lp,
		NextTradeID: 1,
	}, nil
}

// BestBid returns the highest bid price and its total quantity.
func (b *OrderBook) BestBid() (price, qty int64, ok bool) {
	if b == nil || b.bestBid == nil {
		return 0, 0, false
	}
	return b.bestBid.Price, b.bestBid.TotalQty, true
}

// BestAsk returns the lowest ask price and its total quantity.
func (b *OrderBook) BestAsk() (price, qty int64, ok bool) {
	if b == nil || b.bestAsk == nil {
		return 0, 0, false
	}
	return b.bestAsk.Price, b.bestAsk.TotalQty, true
}

// Depth writes up to n levels from the touch into dst.
// Bids walk toward lower prices. Asks walk toward higher prices.
// The call does not allocate and does not mutate the book.
func (b *OrderBook) Depth(side Side, n int, dst []LevelView) []LevelView {
	if cap(dst) == 0 {
		return dst[:0]
	}
	out := dst[:0]
	if b == nil || n <= 0 {
		return out
	}
	var lvl *LimitLevel
	var step func(*LimitLevel) *LimitLevel
	switch side {
	case SideBuy:
		lvl = b.bestBid
		step = b.Bids.Prev
	case SideSell:
		lvl = b.bestAsk
		step = b.Asks.Next
	default:
		return out
	}
	limit := n
	if limit > cap(out) {
		limit = cap(out)
	}
	for lvl != nil && len(out) < limit {
		out = append(out, LevelView{Price: lvl.Price, Qty: lvl.TotalQty, Count: lvl.Count})
		lvl = step(lvl)
	}
	return out
}

func (b *OrderBook) orderByID(id uint64) *Order {
	slot, ok := b.orders.Lookup(id)
	if !ok {
		return nil
	}
	return b.orderPool.At(slot)
}

func (b *OrderBook) bufferOK(dst []events.Event, need int) bool {
	return cap(dst)-len(dst) >= need
}

func (b *OrderBook) cacheBest(side Side, lvl *LimitLevel) {
	if side == SideBuy {
		if b.bestBid == nil || lvl.Price > b.bestBid.Price {
			b.bestBid = lvl
		}
		return
	}
	if b.bestAsk == nil || lvl.Price < b.bestAsk.Price {
		b.bestAsk = lvl
	}
}

func (b *OrderBook) removeLevel(side Side, lvl *LimitLevel) {
	if side == SideBuy {
		wasBest := b.bestBid == lvl
		b.Bids.Delete(lvl)
		if wasBest {
			b.bestBid = b.Bids.Max()
		}
	} else {
		wasBest := b.bestAsk == lvl
		b.Asks.Delete(lvl)
		if wasBest {
			b.bestAsk = b.Asks.Min()
		}
	}
	b.levelPool.Put(lvl)
}

func (b *OrderBook) rollbackLevel(side Side, lvl *LimitLevel) {
	b.removeLevel(side, lvl)
}
