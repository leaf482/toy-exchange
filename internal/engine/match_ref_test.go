package engine

import (
	"testing"

	"github.com/leaf482/toy-exchange/pkg/events"
)

// refBook is a slow, allocating matcher used only from tests.
type refBook struct {
	bids []*refLevel
	asks []*refLevel
	idx  map[uint64]*refOrder
	next uint64
}

type refLevel struct {
	price  int64
	orders []*refOrder
}

type refOrder struct {
	id     uint64
	side   Side
	price  int64
	qty    int64
	leaves int64
	ts     int64
}

func newRef() *refBook {
	return &refBook{idx: make(map[uint64]*refOrder), next: 1}
}

func (r *refBook) Submit(in OrderInput, dst []events.Event) ([]events.Event, error) {
	if cap(dst)-len(dst) < events.MaxEventsFor(int32(len(r.idx))) {
		return dst, ErrBufferFull
	}
	if in.Qty <= 0 {
		return append(dst, reject(in, events.ReasonBadQty)), nil
	}
	if in.Side != SideBuy && in.Side != SideSell {
		return append(dst, reject(in, events.ReasonBadType)), nil
	}
	if in.Type != TypeLimit && in.Type != TypeMarket {
		return append(dst, reject(in, events.ReasonBadType)), nil
	}
	if in.Type == TypeLimit && in.Price <= 0 {
		return append(dst, reject(in, events.ReasonBadPrice)), nil
	}
	if in.ID == 0 || r.idx[in.ID] != nil {
		return append(dst, reject(in, events.ReasonDuplicateID)), nil
	}
	dst = append(dst, events.Event{
		Kind: events.EventAccepted, OrderID: in.ID, Price: in.Price, Qty: in.Qty, Leaves: in.Qty, Timestamp: in.Timestamp,
	})
	taker := &refOrder{id: in.ID, side: in.Side, price: in.Price, qty: in.Qty, leaves: in.Qty, ts: in.Timestamp}
	var last int64
	any := false
	for taker.leaves > 0 {
		lvl := r.opp(taker.side)
		if lvl == nil {
			break
		}
		if in.Type == TypeLimit {
			if taker.side == SideBuy && lvl.price > taker.price {
				break
			}
			if taker.side == SideSell && lvl.price < taker.price {
				break
			}
		}
		maker := lvl.orders[0]
		fill := taker.leaves
		if maker.leaves < fill {
			fill = maker.leaves
		}
		taker.leaves -= fill
		maker.leaves -= fill
		tradeID := r.next
		r.next++
		dst = append(dst, events.Event{
			Kind: events.EventTrade, OrderID: taker.id, MatchID: maker.id, TradeID: tradeID, Price: maker.price, Qty: fill, Timestamp: taker.ts,
		})
		last = fill
		any = true
		if maker.leaves == 0 {
			r.removeOrder(maker)
			dst = append(dst, events.Event{
				Kind: events.EventFilled, OrderID: maker.id, Price: maker.price, Qty: fill, Timestamp: taker.ts,
			})
			continue
		}
		dst = append(dst, events.Event{
			Kind: events.EventPartiallyFilled, OrderID: maker.id, Price: maker.price, Qty: fill, Leaves: maker.leaves, Timestamp: taker.ts,
		})
	}
	if taker.leaves == 0 {
		return append(dst, events.Event{Kind: events.EventFilled, OrderID: taker.id, Price: taker.price, Qty: last, Timestamp: taker.ts}), nil
	}
	if in.Type == TypeMarket {
		return append(dst, events.Event{
			Kind: events.EventCanceled, Reason: events.ReasonIOCRemainder, OrderID: taker.id, Qty: taker.leaves, Leaves: taker.leaves, Timestamp: taker.ts,
		}), nil
	}
	if any {
		dst = append(dst, events.Event{
			Kind: events.EventPartiallyFilled, OrderID: taker.id, Price: taker.price, Qty: last, Leaves: taker.leaves, Timestamp: taker.ts,
		})
	}
	r.rest(taker)
	return dst, nil
}

func (r *refBook) Cancel(id uint64, dst []events.Event) ([]events.Event, error) {
	if cap(dst)-len(dst) < events.MaxEventsFor(int32(len(r.idx))) {
		return dst, ErrBufferFull
	}
	o := r.idx[id]
	if o == nil {
		return dst, ErrNotFound
	}
	leaves := o.leaves
	ts := o.ts
	r.removeOrder(o)
	return append(dst, events.Event{
		Kind: events.EventCanceled, Reason: events.ReasonNone, OrderID: id, Qty: leaves, Leaves: leaves, Timestamp: ts,
	}), nil
}

func (r *refBook) Amend(id uint64, newPrice, newQty, ts int64, dst []events.Event) ([]events.Event, error) {
	if newQty < 0 || newPrice <= 0 {
		if cap(dst)-len(dst) < events.MaxEventsFor(int32(len(r.idx))) {
			return dst, ErrBufferFull
		}
		return dst, ErrBadAmend
	}
	o := r.idx[id]
	if o == nil {
		if cap(dst)-len(dst) < events.MaxEventsFor(int32(len(r.idx))) {
			return dst, ErrBufferFull
		}
		return dst, ErrNotFound
	}
	if newQty == 0 {
		return r.Cancel(id, dst)
	}
	if newPrice == o.price && newQty <= o.leaves {
		if cap(dst)-len(dst) < events.MaxEventsFor(int32(len(r.idx))) {
			return dst, ErrBufferFull
		}
		o.leaves = newQty
		o.ts = ts
		return append(dst, events.Event{
			Kind: events.EventAmended, OrderID: id, Price: o.price, Qty: o.leaves, Leaves: o.leaves, Timestamp: ts,
		}), nil
	}
	if cap(dst)-len(dst) < events.MaxEventsFor(int32(len(r.idx)))+1 {
		return dst, ErrBufferFull
	}
	side := o.side
	dst, err := r.Cancel(id, dst)
	if err != nil {
		return dst, err
	}
	return r.Submit(OrderInput{ID: id, Side: side, Type: TypeLimit, Price: newPrice, Qty: newQty, Timestamp: ts}, dst)
}

func (r *refBook) opp(side Side) *refLevel {
	var best *refLevel
	levels := r.asks
	if side == SideSell {
		levels = r.bids
	}
	for _, lvl := range levels {
		if len(lvl.orders) == 0 {
			continue
		}
		if best == nil {
			best = lvl
			continue
		}
		if side == SideBuy && lvl.price < best.price {
			best = lvl
		}
		if side == SideSell && lvl.price > best.price {
			best = lvl
		}
	}
	return best
}

func (r *refBook) rest(o *refOrder) {
	levels := &r.asks
	if o.side == SideBuy {
		levels = &r.bids
	}
	var lvl *refLevel
	for _, cur := range *levels {
		if cur.price == o.price {
			lvl = cur
			break
		}
	}
	if lvl == nil {
		lvl = &refLevel{price: o.price}
		*levels = append(*levels, lvl)
	}
	lvl.orders = append(lvl.orders, o)
	r.idx[o.id] = o
}

func (r *refBook) removeOrder(o *refOrder) {
	delete(r.idx, o.id)
	levels := &r.asks
	if o.side == SideBuy {
		levels = &r.bids
	}
	for i, lvl := range *levels {
		if lvl.price != o.price {
			continue
		}
		for j, cur := range lvl.orders {
			if cur == o {
				lvl.orders = append(lvl.orders[:j], lvl.orders[j+1:]...)
				break
			}
		}
		if len(lvl.orders) == 0 {
			*levels = append((*levels)[:i], (*levels)[i+1:]...)
		}
		return
	}
}

func TestReferenceMatchesEngine(t *testing.T) {
	const n = 400
	rng := uint64(1)
	next := func() uint64 {
		rng ^= rng << 13
		rng ^= rng >> 7
		rng ^= rng << 17
		return rng
	}
	b := newBook(t, n, 64, n)
	ref := newRef()
	var issued []uint64
	nextID := uint64(1)
	for i := 0; i < n; i++ {
		roll := next() % 10
		var kind string
		var in OrderInput
		var id uint64
		var price, qty int64
		switch {
		case roll < 6:
			kind = "submit"
		case roll < 8:
			kind = "cancel"
		default:
			kind = "amend"
		}
		dstE := evbuf(events.MaxEventsFor(int32(n)))
		dstR := evbuf(events.MaxEventsFor(int32(n)))
		var errE, errR error
		switch kind {
		case "cancel":
			id = 1
			if len(issued) > 0 {
				id = issued[next()%uint64(len(issued))]
			}
			dstE, errE = b.Cancel(id, dstE[:0])
			dstR, errR = ref.Cancel(id, dstR[:0])
		case "amend":
			id = 1
			if len(issued) > 0 {
				id = issued[next()%uint64(len(issued))]
			}
			price = int64(90 + next()%21)
			qty = int64(next() % 8)
			ts := int64(i + 1)
			dstE, errE = b.Amend(id, price, qty, ts, dstE[:0])
			dstR, errR = ref.Amend(id, price, qty, ts, dstR[:0])
		default:
			id = nextID
			nextID++
			issued = append(issued, id)
			side := SideBuy
			px := int64(90 + next()%11)
			if next()%2 == 0 {
				side = SideSell
				px = int64(100 + next()%11)
			}
			typ := TypeLimit
			if next()%5 == 0 {
				typ = TypeMarket
			}
			qty = int64(1 + next()%5)
			in = OrderInput{ID: id, Side: side, Type: typ, Price: px, Qty: qty, Timestamp: int64(i + 1)}
			dstE, errE = b.Submit(in, dstE[:0])
			dstR, errR = ref.Submit(in, dstR[:0])
		}
		if (errE == nil) != (errR == nil) || (errE != nil && errE.Error() != errR.Error()) {
			t.Fatalf("op %d %s errors %v %v", i, kind, errE, errR)
		}
		if formatEvents(dstE) != formatEvents(dstR) {
			t.Fatalf("op %d %s\nengine:\n%s\nref:\n%s", i, kind, formatEvents(dstE), formatEvents(dstR))
		}
		if err := b.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
	}
}
