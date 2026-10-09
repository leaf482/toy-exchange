package engine

import "testing"

func newOrder(id uint64, leaves int64) *Order {
	return &Order{ID: id, Side: SideBuy, Type: TypeLimit, Price: 100, Qty: leaves, Leaves: leaves}
}

func assertQueue(t *testing.T, lvl *LimitLevel, ids ...uint64) {
	t.Helper()
	var sum int64
	var n int32
	seen := map[*Order]bool{}
	var prev *Order
	for o := lvl.Head; o != nil; o = o.Next {
		if seen[o] {
			t.Fatal("cycle")
		}
		seen[o] = true
		if o.Prev != prev {
			t.Fatalf("order %d prev", o.ID)
		}
		if o.Level != lvl {
			t.Fatalf("order %d level", o.ID)
		}
		if n < int32(len(ids)) && o.ID != ids[n] {
			t.Fatalf("position %d id %d, want %d", n, o.ID, ids[n])
		}
		sum += o.Leaves
		n++
		prev = o
	}
	if n != int32(len(ids)) {
		t.Fatalf("len %d, want %d", n, len(ids))
	}
	if prev != lvl.Tail {
		t.Fatal("tail")
	}
	if lvl.Tail != nil && lvl.Tail.Next != nil {
		t.Fatal("tail next")
	}
	if lvl.Count != n || lvl.TotalQty != sum {
		t.Fatalf("count=%d qty=%d, walked %d/%d", lvl.Count, lvl.TotalQty, n, sum)
	}
	if n == 0 && (lvl.Head != nil || lvl.Tail != nil) {
		t.Fatal("empty links")
	}
}

func TestLimitLevelQueue(t *testing.T) {
	lvl := &LimitLevel{Price: 100}
	assertQueue(t, lvl)
	if lvl.PopFront() != nil {
		t.Fatal("pop empty")
	}

	a := newOrder(1, 5)
	lvl.PushBack(a)
	assertQueue(t, lvl, 1)
	if lvl.Head != a || lvl.Tail != a {
		t.Fatal("single head/tail")
	}

	b := newOrder(2, 3)
	c := newOrder(3, 1)
	lvl.PushBack(b)
	lvl.PushBack(c)
	assertQueue(t, lvl, 1, 2, 3)

	lvl.PushBack(b)
	assertQueue(t, lvl, 1, 2, 3)

	lvl.Unlink(b)
	assertQueue(t, lvl, 1, 3)
	if b.Level != nil || b.Prev != nil || b.Next != nil {
		t.Fatal("unlinked order still linked")
	}

	lvl.Unlink(b)
	assertQueue(t, lvl, 1, 3)

	if lvl.PopFront() != a {
		t.Fatal("pop head")
	}
	assertQueue(t, lvl, 3)

	d := newOrder(4, 7)
	lvl.PushBack(d)
	assertQueue(t, lvl, 3, 4)
	lvl.Unlink(d)
	assertQueue(t, lvl, 3)
	if lvl.PopFront() != c {
		t.Fatal("pop last")
	}
	assertQueue(t, lvl)
}

func TestOrderPoolZeroAndSlot(t *testing.T) {
	p := NewOrderPool(2)
	if p == nil || p.Cap() != 2 || p.InUse() != 0 {
		t.Fatal("new pool")
	}
	a, ok := p.Get()
	if !ok || a.Slot != 0 {
		t.Fatalf("slot %d ok %v", a.Slot, ok)
	}
	a.Qty = 9
	a.Leaves = 4
	a.ID = 42
	p.Put(a)
	if p.InUse() != 0 {
		t.Fatal("in use after put")
	}
	b, ok := p.Get()
	if !ok || b != a {
		t.Fatal("slab pointer not reused")
	}
	if b.Qty != 0 || b.Leaves != 0 || b.ID != 0 || b.Slot != 0 {
		t.Fatalf("not zeroed: %+v", *b)
	}

	stack := &Order{Slot: b.Slot}
	p.Put(stack)
	if p.InUse() != 1 {
		t.Fatal("foreign put")
	}
	if NewOrderPool(0) != nil {
		t.Fatal("zero cap")
	}

	levels := NewLevelPool(1)
	lvl, ok := levels.Get()
	if !ok || lvl.Slot != 0 {
		t.Fatal("level get")
	}
	lvl.Price = 7
	lvl.TotalQty = 3
	levels.Put(lvl)
	lvl2, ok := levels.Get()
	if !ok || lvl2.Price != 0 || lvl2.TotalQty != 0 || lvl2.Slot != 0 {
		t.Fatalf("level not zeroed: %+v", *lvl2)
	}
}

func TestOrderPoolNoAlloc(t *testing.T) {
	p := NewOrderPool(4)
	o, ok := p.Get()
	if !ok {
		t.Fatal("warmup")
	}
	p.Put(o)
	allocs := testing.AllocsPerRun(200, func() {
		o, ok := p.Get()
		if ok {
			p.Put(o)
		}
	})
	if allocs != 0 {
		t.Fatalf("allocs/op = %v", allocs)
	}
}
