package index

import "testing"

func TestOpenInsertDeleteReuse(t *testing.T) {
	o := NewOpen(8)
	if o == nil || o.Len() != 0 {
		t.Fatal("new")
	}
	if o.Insert(0, 1) {
		t.Fatal("id 0")
	}
	for i := uint64(1); i <= 8; i++ {
		if !o.Insert(i, uint32(i)) {
			t.Fatalf("insert %d", i)
		}
	}
	if o.Insert(3, 9) {
		t.Fatal("duplicate")
	}
	slot, ok := o.Lookup(3)
	if !ok || slot != 3 {
		t.Fatalf("lookup %d %v", slot, ok)
	}
	o.Delete(3)
	if o.Has(3) || o.Len() != 7 {
		t.Fatal("deleted")
	}
	if !o.Insert(3, 30) {
		t.Fatal("reuse tomb")
	}
	slot, ok = o.Lookup(3)
	if !ok || slot != 30 {
		t.Fatalf("reinserted %d", slot)
	}
	for i := uint64(1); i <= 8; i++ {
		o.Delete(i)
	}
	if o.Len() != 0 || o.Has(1) {
		t.Fatal("empty")
	}
	if !o.Insert(1, 0) {
		t.Fatal("insert after wipe, slot 0")
	}
	slot, ok = o.Lookup(1)
	if !ok || slot != 0 {
		t.Fatal("slot 0 lookup")
	}
	if NewOpen(0) != nil {
		t.Fatal("cap")
	}
}

func TestOpenLoadLimit(t *testing.T) {
	o := NewOpen(4)
	var n int
	for i := uint64(1); i < 10000; i++ {
		if !o.Insert(i, uint32(i)) {
			break
		}
		n++
	}
	if n == 0 || o.Len() != n {
		t.Fatalf("len %d n %d", o.Len(), n)
	}
	if o.Insert(99999, 1) {
		t.Fatal("accepted past load")
	}
}

func TestOpenRandom(t *testing.T) {
	const n = 500
	o := NewOpen(n)
	rng := uint64(1)
	next := func() uint64 {
		rng ^= rng << 13
		rng ^= rng >> 7
		rng ^= rng << 17
		return rng
	}
	live := map[uint64]uint32{}
	for step := 0; step < n*3; step++ {
		id := 1 + next()%uint64(n*2)
		if next()%3 == 0 {
			o.Delete(id)
			delete(live, id)
		} else if _, ok := live[id]; ok {
			if o.Insert(id, 1) {
				t.Fatal("duplicate accepted")
			}
		} else if o.Insert(id, uint32(id)) {
			live[id] = uint32(id)
		}
		if o.Len() != len(live) {
			t.Fatalf("len %d map %d", o.Len(), len(live))
		}
		for id, slot := range live {
			got, ok := o.Lookup(id)
			if !ok || got != slot {
				t.Fatalf("missing %d", id)
			}
		}
	}
}

func TestOpenNoAlloc(t *testing.T) {
	o := NewOpen(32)
	if !o.Insert(1, 1) {
		t.Fatal("warmup")
	}
	o.Delete(1)
	allocs := testing.AllocsPerRun(200, func() {
		if o.Insert(1, 1) {
			o.Delete(1)
		}
	})
	if allocs != 0 {
		t.Fatalf("allocs/op = %v", allocs)
	}
}
