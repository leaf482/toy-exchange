package engine

import "fmt"

// CheckInvariants reports the first broken book invariant.
func (b *OrderBook) CheckInvariants() error {
	if b == nil {
		return fmt.Errorf("nil book")
	}
	if b.NextTradeID < 1 {
		return fmt.Errorf("next trade id %d", b.NextTradeID)
	}
	seen := make(map[*Order]struct{})
	if err := b.walkSide(SideBuy, b.bestBid, b.Bids.Max(), seen); err != nil {
		return err
	}
	if err := b.walkSide(SideSell, b.bestAsk, b.Asks.Min(), seen); err != nil {
		return err
	}
	if b.bestBid != nil && b.bestAsk != nil && b.bestBid.Price >= b.bestAsk.Price {
		return fmt.Errorf("crossed book %d >= %d", b.bestBid.Price, b.bestAsk.Price)
	}
	if int(b.RestingCount) != len(seen) || b.orders.Len() != len(seen) {
		return fmt.Errorf("resting %d index %d walked %d", b.RestingCount, b.orders.Len(), len(seen))
	}
	var indexed int
	var indexErr error
	b.orders.Range(func(id uint64, slot uint32) {
		if indexErr != nil {
			return
		}
		o := b.orderPool.At(slot)
		if o == nil || o.ID != id {
			indexErr = fmt.Errorf("index id %d slot %d", id, slot)
			return
		}
		if _, ok := seen[o]; !ok {
			indexErr = fmt.Errorf("index order %d is not resting", id)
			return
		}
		indexed++
	})
	if indexErr != nil {
		return indexErr
	}
	if indexed != len(seen) {
		return fmt.Errorf("index visits %d, resting %d", indexed, len(seen))
	}
	if b.orderPool.InUse() != len(seen) {
		return fmt.Errorf("order pool in use %d, resting %d", b.orderPool.InUse(), len(seen))
	}
	return nil
}

func (b *OrderBook) walkSide(side Side, cached, extreme *LimitLevel, seen map[*Order]struct{}) error {
	if cached != extreme {
		return fmt.Errorf("side %d best cache mismatch", side)
	}
	var root *LimitLevel
	if side == SideBuy {
		root = b.Bids.Root
	} else {
		root = b.Asks.Root
	}
	var walk func(n *LimitLevel) error
	walk = func(n *LimitLevel) error {
		if n == nil {
			return nil
		}
		if err := walk(n.Left); err != nil {
			return err
		}
		if err := checkLevel(side, n, seen); err != nil {
			return err
		}
		return walk(n.Right)
	}
	return walk(root)
}

func checkLevel(side Side, lvl *LimitLevel, seen map[*Order]struct{}) error {
	var sum int64
	var n int32
	var prev *Order
	for o := lvl.Head; o != nil; o = o.Next {
		if o.Prev != prev {
			return fmt.Errorf("price %d order %d prev", lvl.Price, o.ID)
		}
		if o.Level != lvl {
			return fmt.Errorf("order %d level", o.ID)
		}
		if o.Side != side || o.Type != TypeLimit {
			return fmt.Errorf("order %d side/type", o.ID)
		}
		if _, ok := seen[o]; ok {
			return fmt.Errorf("order %d in two levels", o.ID)
		}
		seen[o] = struct{}{}
		sum += o.Leaves
		n++
		prev = o
	}
	if prev != lvl.Tail {
		return fmt.Errorf("price %d tail", lvl.Price)
	}
	if lvl.Tail != nil && lvl.Tail.Next != nil {
		return fmt.Errorf("price %d tail next", lvl.Price)
	}
	if n != lvl.Count || sum != lvl.TotalQty {
		return fmt.Errorf("price %d count %d/%d qty %d/%d", lvl.Price, lvl.Count, n, lvl.TotalQty, sum)
	}
	if n == 0 {
		return fmt.Errorf("empty level %d still in tree", lvl.Price)
	}
	return nil
}
