package engine

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/leaf482/toy-exchange/pkg/events"
)

func formatEvents(ev []events.Event) string {
	var b strings.Builder
	for _, e := range ev {
		switch e.Kind {
		case events.EventTrade:
			fmt.Fprintf(&b, "TRADE %d %d %d %d %d %d\n", e.TradeID, e.OrderID, e.MatchID, e.Price, e.Qty, e.Timestamp)
		case events.EventAccepted:
			fmt.Fprintf(&b, "ACCEPTED %d %d %d\n", e.OrderID, e.Leaves, e.Timestamp)
		case events.EventPartiallyFilled:
			fmt.Fprintf(&b, "PARTIAL %d %d %d %d\n", e.OrderID, e.Qty, e.Leaves, e.Timestamp)
		case events.EventFilled:
			fmt.Fprintf(&b, "FILLED %d %d %d\n", e.OrderID, e.Qty, e.Timestamp)
		case events.EventCanceled:
			fmt.Fprintf(&b, "CANCELED %d %d %d %d\n", e.OrderID, e.Leaves, e.Reason, e.Timestamp)
		case events.EventRejected:
			fmt.Fprintf(&b, "REJECTED %d %d %d\n", e.OrderID, e.Reason, e.Timestamp)
		case events.EventAmended:
			fmt.Fprintf(&b, "AMENDED %d %d %d %d\n", e.OrderID, e.Price, e.Leaves, e.Timestamp)
		default:
			fmt.Fprintf(&b, "UNKNOWN %d\n", e.Kind)
		}
	}
	return b.String()
}

func evbuf(n int) []events.Event {
	if n < 64 {
		n = 64
	}
	return make([]events.Event, 0, n)
}

func newBook(t *testing.T, orders, levels, index int) *OrderBook {
	t.Helper()
	b, err := New(orders, levels, index)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func parseFixture(t *testing.T, path string) []OrderInput {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []OrderInput
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		kind := f[0]
		side := SideBuy
		if f[1] == "S" {
			side = SideSell
		}
		id, _ := strconv.ParseUint(f[2], 10, 64)
		switch kind {
		case "L":
			price, _ := strconv.ParseInt(f[3], 10, 64)
			qty, _ := strconv.ParseInt(f[4], 10, 64)
			ts, _ := strconv.ParseInt(f[5], 10, 64)
			out = append(out, OrderInput{ID: id, Side: side, Type: TypeLimit, Price: price, Qty: qty, Timestamp: ts})
		case "M":
			qty, _ := strconv.ParseInt(f[3], 10, 64)
			ts, _ := strconv.ParseInt(f[4], 10, 64)
			out = append(out, OrderInput{ID: id, Side: side, Type: TypeMarket, Qty: qty, Timestamp: ts})
		default:
			t.Fatalf("bad line %q", line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func runInputs(t *testing.T, b *OrderBook, in []OrderInput) string {
	t.Helper()
	dst := evbuf(4096)
	var all []events.Event
	var trades uint64
	for _, op := range in {
		var err error
		dst, err = b.Submit(op, dst[:0])
		mustOK(t, err)
		all = append(all, dst...)
		for _, e := range dst {
			if e.Kind == events.EventTrade {
				trades++
			}
		}
		if b.NextTradeID != trades+1 {
			t.Fatalf("trade id %d after %d trades", b.NextTradeID, trades)
		}
		if err := b.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
	}
	return formatEvents(all)
}

func idsAt(b *OrderBook, side Side, price int64) []uint64 {
	var tree *PriceTree
	if side == SideBuy {
		tree = &b.Bids
	} else {
		tree = &b.Asks
	}
	lvl := tree.Find(price)
	if lvl == nil {
		return nil
	}
	var ids []uint64
	for o := lvl.Head; o != nil; o = o.Next {
		ids = append(ids, o.ID)
	}
	return ids
}
