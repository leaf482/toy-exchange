package engine

import "github.com/leaf482/toy-exchange/pkg/events"

// Submit validates an order, matches it, and rests any limit remainder.
// Validation failures append EventRejected and return a nil error.
// ErrBufferFull is returned before any book change.
func (b *OrderBook) Submit(in OrderInput, dst []events.Event) ([]events.Event, error) {
	if b == nil {
		return dst, ErrCapacity
	}
	if !b.bufferOK(dst, events.MaxEventsFor(b.RestingCount)) {
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
	if in.ID == 0 || b.orders.Has(in.ID) {
		return append(dst, reject(in, events.ReasonDuplicateID)), nil
	}

	dst = append(dst, events.Event{
		Kind:      events.EventAccepted,
		OrderID:   in.ID,
		Price:     in.Price,
		Qty:       in.Qty,
		Leaves:    in.Qty,
		Timestamp: in.Timestamp,
	})
	taker := Order{
		ID:        in.ID,
		Side:      in.Side,
		Type:      in.Type,
		Price:     in.Price,
		Qty:       in.Qty,
		Leaves:    in.Qty,
		Timestamp: in.Timestamp,
	}
	var last int64
	var any bool
	dst, last, any = b.sweep(dst, &taker)
	if taker.Leaves == 0 {
		return append(dst, events.Event{
			Kind:      events.EventFilled,
			OrderID:   taker.ID,
			Price:     taker.Price,
			Qty:       last,
			Leaves:    0,
			Timestamp: taker.Timestamp,
		}), nil
	}
	if taker.Type == TypeMarket {
		return append(dst, events.Event{
			Kind:      events.EventCanceled,
			Reason:    events.ReasonIOCRemainder,
			OrderID:   taker.ID,
			Qty:       taker.Leaves,
			Leaves:    taker.Leaves,
			Timestamp: taker.Timestamp,
		}), nil
	}
	if any {
		dst = append(dst, events.Event{
			Kind:      events.EventPartiallyFilled,
			OrderID:   taker.ID,
			Price:     taker.Price,
			Qty:       last,
			Leaves:    taker.Leaves,
			Timestamp: taker.Timestamp,
		})
	}
	if !b.rest(&taker) {
		dst = append(dst, events.Event{
			Kind:      events.EventCanceled,
			Reason:    events.ReasonPoolExhausted,
			OrderID:   taker.ID,
			Qty:       taker.Leaves,
			Leaves:    taker.Leaves,
			Timestamp: taker.Timestamp,
		})
	}
	return dst, nil
}

// Cancel removes a resting order. An unknown id returns ErrNotFound.
func (b *OrderBook) Cancel(id uint64, dst []events.Event) ([]events.Event, error) {
	if b == nil {
		return dst, ErrCapacity
	}
	if !b.bufferOK(dst, events.MaxEventsFor(b.RestingCount)) {
		return dst, ErrBufferFull
	}
	o := b.orderByID(id)
	if o == nil {
		return dst, ErrNotFound
	}
	return b.cancelFound(o, dst, events.ReasonNone), nil
}

// Amend decreases size in place, or cancels and resubmits when the price
// changes or the resting size would grow.
func (b *OrderBook) Amend(id uint64, newPrice, newQty, ts int64, dst []events.Event) ([]events.Event, error) {
	if b == nil {
		return dst, ErrCapacity
	}
	if newQty < 0 || newPrice <= 0 {
		if !b.bufferOK(dst, events.MaxEventsFor(b.RestingCount)) {
			return dst, ErrBufferFull
		}
		return dst, ErrBadAmend
	}
	o := b.orderByID(id)
	if o == nil {
		if !b.bufferOK(dst, events.MaxEventsFor(b.RestingCount)) {
			return dst, ErrBufferFull
		}
		return dst, ErrNotFound
	}
	if newQty == 0 {
		return b.Cancel(id, dst)
	}
	if newPrice == o.Price && newQty <= o.Leaves {
		if !b.bufferOK(dst, events.MaxEventsFor(b.RestingCount)) {
			return dst, ErrBufferFull
		}
		delta := o.Leaves - newQty
		o.Leaves = newQty
		o.Level.TotalQty -= delta
		o.Timestamp = ts
		return append(dst, events.Event{
			Kind:      events.EventAmended,
			OrderID:   id,
			Price:     o.Price,
			Qty:       o.Leaves,
			Leaves:    o.Leaves,
			Timestamp: ts,
		}), nil
	}
	if !b.bufferOK(dst, events.MaxEventsFor(b.RestingCount)+1) {
		return dst, ErrBufferFull
	}
	side := o.Side
	dst = b.cancelFound(o, dst, events.ReasonNone)
	return b.Submit(OrderInput{
		ID:        id,
		Side:      side,
		Type:      TypeLimit,
		Price:     newPrice,
		Qty:       newQty,
		Timestamp: ts,
	}, dst)
}

func reject(in OrderInput, reason events.Reason) events.Event {
	return events.Event{
		Kind:      events.EventRejected,
		Reason:    reason,
		OrderID:   in.ID,
		Price:     in.Price,
		Qty:       in.Qty,
		Leaves:    in.Qty,
		Timestamp: in.Timestamp,
	}
}

func (b *OrderBook) sweep(dst []events.Event, taker *Order) ([]events.Event, int64, bool) {
	var last int64
	any := false
	for taker.Leaves > 0 {
		var lvl *LimitLevel
		var makerSide Side
		if taker.Side == SideBuy {
			lvl = b.bestAsk
			makerSide = SideSell
			if lvl == nil || (taker.Type == TypeLimit && lvl.Price > taker.Price) {
				break
			}
		} else {
			lvl = b.bestBid
			makerSide = SideBuy
			if lvl == nil || (taker.Type == TypeLimit && lvl.Price < taker.Price) {
				break
			}
		}
		maker := lvl.Head
		if maker == nil || maker.Leaves <= 0 {
			break
		}
		fill := taker.Leaves
		if maker.Leaves < fill {
			fill = maker.Leaves
		}
		taker.Leaves -= fill
		maker.Leaves -= fill
		lvl.TotalQty -= fill
		tradeID := b.NextTradeID
		b.NextTradeID++
		dst = append(dst, events.Event{
			Kind:      events.EventTrade,
			OrderID:   taker.ID,
			MatchID:   maker.ID,
			TradeID:   tradeID,
			Price:     maker.Price,
			Qty:       fill,
			Timestamp: taker.Timestamp,
		})
		last = fill
		any = true
		if maker.Leaves == 0 {
			id := maker.ID
			price := maker.Price
			b.finishMaker(maker, lvl, makerSide)
			dst = append(dst, events.Event{
				Kind:      events.EventFilled,
				OrderID:   id,
				Price:     price,
				Qty:       fill,
				Leaves:    0,
				Timestamp: taker.Timestamp,
			})
			continue
		}
		dst = append(dst, events.Event{
			Kind:      events.EventPartiallyFilled,
			OrderID:   maker.ID,
			Price:     maker.Price,
			Qty:       fill,
			Leaves:    maker.Leaves,
			Timestamp: taker.Timestamp,
		})
	}
	return dst, last, any
}

func (b *OrderBook) finishMaker(maker *Order, lvl *LimitLevel, side Side) {
	b.orders.Delete(maker.ID)
	lvl.Unlink(maker)
	b.RestingCount--
	if lvl.Count == 0 {
		b.removeLevel(side, lvl)
	}
	b.orderPool.Put(maker)
}

func (b *OrderBook) cancelFound(o *Order, dst []events.Event, reason events.Reason) []events.Event {
	id := o.ID
	leaves := o.Leaves
	ts := o.Timestamp
	side := o.Side
	lvl := o.Level
	b.orders.Delete(id)
	lvl.Unlink(o)
	b.RestingCount--
	if lvl.Count == 0 {
		b.removeLevel(side, lvl)
	}
	dst = append(dst, events.Event{
		Kind:      events.EventCanceled,
		Reason:    reason,
		OrderID:   id,
		Qty:       leaves,
		Leaves:    leaves,
		Timestamp: ts,
	})
	b.orderPool.Put(o)
	return dst
}

func (b *OrderBook) rest(taker *Order) bool {
	var tree *PriceTree
	if taker.Side == SideBuy {
		tree = &b.Bids
	} else {
		tree = &b.Asks
	}
	lvl := tree.Find(taker.Price)
	fresh := false
	if lvl == nil {
		var ok bool
		lvl, ok = b.levelPool.Get()
		if !ok {
			return false
		}
		lvl.Price = taker.Price
		if !tree.Insert(lvl) {
			b.levelPool.Put(lvl)
			return false
		}
		fresh = true
	}
	slot, ok := b.orderPool.Get()
	if !ok {
		if fresh {
			b.rollbackLevel(taker.Side, lvl)
		}
		return false
	}
	slotNo := slot.Slot
	if !b.orders.Insert(taker.ID, slotNo) {
		b.orderPool.Put(slot)
		if fresh {
			b.rollbackLevel(taker.Side, lvl)
		}
		return false
	}
	*slot = Order{
		ID:        taker.ID,
		Side:      taker.Side,
		Type:      TypeLimit,
		Price:     taker.Price,
		Qty:       taker.Qty,
		Leaves:    taker.Leaves,
		Timestamp: taker.Timestamp,
		Slot:      slotNo,
	}
	lvl.PushBack(slot)
	b.RestingCount++
	b.cacheBest(taker.Side, lvl)
	return true
}
