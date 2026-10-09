package engine

import "github.com/leaf482/toy-exchange/pkg/events"

// OpKind is one step in a prebuilt benchmark script.
type OpKind uint8

const (
	OpLimit OpKind = iota
	OpCancel
	OpMarket
)

// Op is a single scripted command. The benchmark builds the slice before the timer.
type Op struct {
	Kind     OpKind
	In       OrderInput
	CancelID uint64
}

// BuildScript returns n deterministic operations.
// Seed 1 is the benchmark script. The mix is 70% limit, 20% cancel, 10% market.
// Timestamps are the zero-based op index.
func BuildScript(n int, seed uint64) []Op {
	if n <= 0 {
		return nil
	}
	rng := seed
	next := func() uint64 {
		rng ^= rng << 13
		rng ^= rng >> 7
		rng ^= rng << 17
		return rng
	}
	const mid int64 = 10_000
	out := make([]Op, n)
	var nextID uint64 = 1
	var issued uint64
	for i := 0; i < n; i++ {
		roll := next() % 10
		ts := int64(i)
		switch {
		case roll < 7:
			side := SideBuy
			price := mid - 16 + int64(next()%17)
			if next()%2 == 1 {
				side = SideSell
				price = mid + int64(next()%17)
			}
			out[i] = Op{Kind: OpLimit, In: OrderInput{
				ID: nextID, Side: side, Type: TypeLimit, Price: price, Qty: 1 + int64(next()%10), Timestamp: ts,
			}}
			nextID++
			issued++
		case roll < 9:
			id := uint64(1)
			if issued > 0 {
				id = 1 + next()%issued
			}
			out[i] = Op{Kind: OpCancel, CancelID: id, In: OrderInput{Timestamp: ts}}
		default:
			side := SideBuy
			if next()%2 == 1 {
				side = SideSell
			}
			out[i] = Op{Kind: OpMarket, In: OrderInput{
				ID: nextID, Side: side, Type: TypeMarket, Qty: 1 + int64(next()%10), Timestamp: ts,
			}}
			nextID++
			issued++
		}
	}
	return out
}

// Apply runs one script op. A missing cancel id returns ErrNotFound.
func (b *OrderBook) Apply(op Op, dst []events.Event) ([]events.Event, error) {
	switch op.Kind {
	case OpCancel:
		return b.Cancel(op.CancelID, dst)
	default:
		return b.Submit(op.In, dst)
	}
}
