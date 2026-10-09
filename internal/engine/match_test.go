package engine

import (
	"os"
	"testing"

	"github.com/leaf482/toy-exchange/pkg/events"
)

func TestBasicFIFOGolden(t *testing.T) {
	b := newBook(t, 16, 16, 16)
	in := parseFixture(t, "../../test/fixtures/basic_fifo.txt")
	got := runInputs(t, b, in)
	want, err := os.ReadFile("../../test/fixtures/basic_fifo.out.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
	if _, _, ok := b.BestBid(); ok {
		t.Fatal("book should be empty")
	}
	if _, _, ok := b.BestAsk(); ok {
		t.Fatal("ask should be empty")
	}
}

func TestLimitWalksLevelsAndRests(t *testing.T) {
	b := newBook(t, 16, 16, 16)
	dst := evbuf(0)
	var err error
	dst, err = b.Submit(OrderInput{ID: 1, Side: SideSell, Type: TypeLimit, Price: 101, Qty: 2, Timestamp: 1}, dst[:0])
	mustOK(t, err)
	dst, err = b.Submit(OrderInput{ID: 2, Side: SideSell, Type: TypeLimit, Price: 103, Qty: 2, Timestamp: 2}, dst[:0])
	mustOK(t, err)
	dst, err = b.Submit(OrderInput{ID: 3, Side: SideBuy, Type: TypeLimit, Price: 100, Qty: 1, Timestamp: 3}, dst[:0])
	mustOK(t, err)
	if price, _, ok := b.BestBid(); !ok || price != 100 {
		t.Fatalf("bid %d %v", price, ok)
	}
	if price, qty, ok := b.BestAsk(); !ok || price != 101 || qty != 2 {
		t.Fatalf("ask %d %d", price, qty)
	}

	dst, err = b.Submit(OrderInput{ID: 4, Side: SideBuy, Type: TypeLimit, Price: 104, Qty: 3, Timestamp: 4}, dst[:0])
	mustOK(t, err)
	log := formatEvents(dst)
	want := "" +
		"ACCEPTED 4 3 4\n" +
		"TRADE 1 4 1 101 2 4\n" +
		"FILLED 1 2 4\n" +
		"TRADE 2 4 2 103 1 4\n" +
		"PARTIAL 2 1 1 4\n" +
		"FILLED 4 1 4\n"
	if log != want {
		t.Fatalf("log\n%s", log)
	}
	if price, qty, ok := b.BestAsk(); !ok || price != 103 || qty != 1 {
		t.Fatalf("ask after %d %d %v", price, qty, ok)
	}
	if price, qty, ok := b.BestBid(); !ok || price != 100 || qty != 1 {
		t.Fatalf("bid after %d %d", price, qty)
	}
	if err := b.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestDepthAndCancelMiddle(t *testing.T) {
	b := newBook(t, 16, 16, 16)
	dst := evbuf(0)
	for i, px := range []int64{98, 100, 99} {
		var err error
		dst, err = b.Submit(OrderInput{ID: uint64(i + 1), Side: SideBuy, Type: TypeLimit, Price: px, Qty: 1, Timestamp: int64(i + 1)}, dst[:0])
		mustOK(t, err)
	}
	depth := b.Depth(SideBuy, 2, make([]LevelView, 3))
	if len(depth) != 2 || depth[0].Price != 100 || depth[1].Price != 99 {
		t.Fatalf("depth %+v", depth)
	}
	if len(b.Depth(SideBuy, 10, make([]LevelView, 0))) != 0 {
		t.Fatal("zero cap")
	}

	dst = evbuf(0)
	var err error
	dst, err = b.Submit(OrderInput{ID: 4, Side: SideBuy, Type: TypeLimit, Price: 100, Qty: 2, Timestamp: 4}, dst[:0])
	mustOK(t, err)
	dst, err = b.Submit(OrderInput{ID: 5, Side: SideBuy, Type: TypeLimit, Price: 100, Qty: 3, Timestamp: 5}, dst[:0])
	mustOK(t, err)
	if got := idsAt(b, SideBuy, 100); len(got) != 3 || got[0] != 2 || got[1] != 4 || got[2] != 5 {
		t.Fatalf("queue %v", got)
	}
	dst, err = b.Cancel(4, dst[:0])
	mustOK(t, err)
	if got := idsAt(b, SideBuy, 100); len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Fatalf("after cancel %v", got)
	}
	if _, err := b.Cancel(4, dst[:0]); err != ErrNotFound {
		t.Fatal(err)
	}
	if err := b.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestAmendPriority(t *testing.T) {
	b := newBook(t, 16, 16, 16)
	dst := evbuf(0)
	var err error
	for _, in := range []OrderInput{
		{ID: 1, Side: SideBuy, Type: TypeLimit, Price: 50, Qty: 5, Timestamp: 1},
		{ID: 2, Side: SideBuy, Type: TypeLimit, Price: 50, Qty: 5, Timestamp: 2},
	} {
		dst, err = b.Submit(in, dst[:0])
		mustOK(t, err)
	}
	dst, err = b.Amend(1, 50, 3, 3, dst[:0])
	mustOK(t, err)
	if got := idsAt(b, SideBuy, 50); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("size down %v", got)
	}
	if _, qty, ok := b.BestBid(); !ok || qty != 8 {
		t.Fatalf("qty %d", qty)
	}
	dst, err = b.Amend(1, 50, 9, 4, dst[:0])
	mustOK(t, err)
	if got := idsAt(b, SideBuy, 50); len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Fatalf("size up %v", got)
	}
	dst, err = b.Amend(1, 40, 9, 5, dst[:0])
	mustOK(t, err)
	if got := idsAt(b, SideBuy, 50); len(got) != 1 || got[0] != 2 {
		t.Fatalf("old price %v", got)
	}
	if got := idsAt(b, SideBuy, 40); len(got) != 1 || got[0] != 1 {
		t.Fatalf("new price %v", got)
	}
	if _, err := b.Amend(9, 40, 1, 6, dst[:0]); err != ErrNotFound {
		t.Fatal(err)
	}
	if _, err := b.Amend(2, 0, 1, 7, dst[:0]); err != ErrBadAmend {
		t.Fatal(err)
	}
	if _, err := b.Amend(2, 50, -1, 7, dst[:0]); err != ErrBadAmend {
		t.Fatal(err)
	}
	if err := b.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestMarketIOC(t *testing.T) {
	b := newBook(t, 8, 8, 8)
	dst := evbuf(0)
	dst, err := b.Submit(OrderInput{ID: 1, Side: SideSell, Type: TypeMarket, Qty: 4, Timestamp: 1}, dst[:0])
	mustOK(t, err)
	log := formatEvents(dst)
	if log != "ACCEPTED 1 4 1\nCANCELED 1 4 6 1\n" {
		t.Fatalf("empty book\n%s", log)
	}
	if b.RestingCount != 0 {
		t.Fatal("market rested")
	}

	dst, err = b.Submit(OrderInput{ID: 2, Side: SideBuy, Type: TypeLimit, Price: 10, Qty: 3, Timestamp: 2}, dst[:0])
	mustOK(t, err)
	dst, err = b.Submit(OrderInput{ID: 3, Side: SideSell, Type: TypeMarket, Price: -5, Qty: 3, Timestamp: 3}, dst[:0])
	mustOK(t, err)
	if _, _, ok := b.BestBid(); ok {
		t.Fatal("bid remains")
	}
	if b.RestingCount != 0 {
		t.Fatal("resting")
	}
}

func TestRejectsAndBuffer(t *testing.T) {
	b := newBook(t, 8, 8, 8)
	dst := evbuf(0)
	dst, err := b.Submit(OrderInput{ID: 1, Side: SideBuy, Type: TypeLimit, Price: 5, Qty: 1, Timestamp: 1}, dst[:0])
	mustOK(t, err)
	before := b.RestingCount
	if _, err := b.Submit(OrderInput{ID: 2, Side: SideBuy, Type: TypeLimit, Price: 5, Qty: 1, Timestamp: 2}, nil); err != ErrBufferFull {
		t.Fatal(err)
	}
	if b.RestingCount != before {
		t.Fatal("mutated on short buffer")
	}
	cases := []OrderInput{
		{ID: 2, Side: SideBuy, Type: TypeLimit, Price: 5, Qty: 0, Timestamp: 2},
		{ID: 2, Side: SideBuy, Type: TypeLimit, Price: 0, Qty: 1, Timestamp: 2},
		{ID: 1, Side: SideBuy, Type: TypeLimit, Price: 5, Qty: 1, Timestamp: 2},
		{ID: 2, Side: 9, Type: TypeLimit, Price: 5, Qty: 1, Timestamp: 2},
	}
	reasons := []events.Reason{events.ReasonBadQty, events.ReasonBadPrice, events.ReasonDuplicateID, events.ReasonBadType}
	for i, in := range cases {
		dst, err = b.Submit(in, dst[:0])
		mustOK(t, err)
		if len(dst) != 1 || dst[0].Kind != events.EventRejected || dst[0].Reason != reasons[i] {
			t.Fatalf("case %d %+v", i, dst)
		}
	}
	if err := b.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestPoolExhaustLeavesFills(t *testing.T) {
	b := newBook(t, 1, 4, 4)
	dst := evbuf(0)
	var err error
	dst, err = b.Submit(OrderInput{ID: 1, Side: SideBuy, Type: TypeLimit, Price: 10, Qty: 2, Timestamp: 1}, dst[:0])
	mustOK(t, err)
	dst, err = b.Submit(OrderInput{ID: 2, Side: SideSell, Type: TypeLimit, Price: 11, Qty: 2, Timestamp: 2}, dst[:0])
	mustOK(t, err)
	log := formatEvents(dst)
	if log != "ACCEPTED 2 2 2\nCANCELED 2 2 5 2\n" {
		t.Fatalf("exhaust\n%s", log)
	}
	if price, qty, ok := b.BestBid(); !ok || price != 10 || qty != 2 {
		t.Fatalf("bid survived %d %d %v", price, qty, ok)
	}
	if err := b.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}
