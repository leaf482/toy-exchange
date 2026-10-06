package replay

import "sort"

// ActionKind is one synthetic command derived from an L2 quantity change.
type ActionKind uint8

const (
	// ActionLimit rests a new order for the full quantity at a price.
	ActionLimit ActionKind = iota
	// ActionCancel removes the previous synthetic order at a price.
	ActionCancel
)

// Side values match the matching engine's side numbers.
const (
	SideBuy  int8 = 1
	SideSell int8 = 2
)

// Action is one limit or cancel. Synthesis does not call the matching engine.
type Action struct {
	Kind     ActionKind
	ID       uint64
	CancelID uint64
	Side     int8
	Price    int64
	Qty      int64
	Time     int64
}

// Synth remembers the synthetic order id resting at each price.
// A later diff cancels that id when the quantity changes or the level disappears.
type Synth struct {
	next   uint64
	bidID  map[int64]uint64
	askID  map[int64]uint64
	bidQty map[int64]int64
	askQty map[int64]int64
}

// NewSynth starts order ids at 1.
func NewSynth() *Synth {
	return &Synth{
		next:   1,
		bidID:  make(map[int64]uint64),
		askID:  make(map[int64]uint64),
		bidQty: make(map[int64]int64),
		askQty: make(map[int64]int64),
	}
}

// Push returns the actions that turn the previously pushed book into bids and asks.
// Bids are emitted from high price to low, then asks from low price to high.
// A changed level cancels the previous synthetic id before the replacement limit.
// An unchanged level emits nothing.
func (s *Synth) Push(bids, asks map[int64]int64, ts int64) []Action {
	if s == nil {
		return nil
	}
	var out []Action
	out = s.diffSide(out, SideBuy, s.bidQty, s.bidID, bids, false, ts)
	out = s.diffSide(out, SideSell, s.askQty, s.askID, asks, true, ts)
	return out
}

func (s *Synth) diffSide(out []Action, side int8, qty map[int64]int64, ids map[int64]uint64, next map[int64]int64, asc bool, ts int64) []Action {
	prices := unionPrices(qty, next)
	sort.Slice(prices, func(i, j int) bool {
		if asc {
			return prices[i] < prices[j]
		}
		return prices[i] > prices[j]
	})
	for _, price := range prices {
		oldQ := qty[price]
		newQ := next[price]
		if oldQ == newQ {
			continue
		}
		if oldQ > 0 {
			out = append(out, Action{Kind: ActionCancel, CancelID: ids[price], Time: ts})
			delete(ids, price)
			delete(qty, price)
		}
		if newQ > 0 {
			id := s.next
			s.next++
			ids[price] = id
			qty[price] = newQ
			out = append(out, Action{
				Kind:  ActionLimit,
				ID:    id,
				Side:  side,
				Price: price,
				Qty:   newQ,
				Time:  ts,
			})
		}
	}
	return out
}

func unionPrices(a, b map[int64]int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	for price, qty := range a {
		if qty != 0 {
			seen[price] = struct{}{}
		}
	}
	for price, qty := range b {
		if qty != 0 {
			seen[price] = struct{}{}
		}
	}
	out := make([]int64, 0, len(seen))
	for price := range seen {
		out = append(out, price)
	}
	return out
}
