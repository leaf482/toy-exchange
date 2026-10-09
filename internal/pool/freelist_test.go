package pool

import "testing"

func TestFreeListGetPut(t *testing.T) {
	f := New[int](3)
	if f == nil || f.Cap() != 3 || f.Available() != 3 || f.InUse() != 0 {
		t.Fatalf("new: cap=%d avail=%d inuse=%d", f.Cap(), f.Available(), f.InUse())
	}

	var got []uint32
	for i := 0; i < 3; i++ {
		item, slot, ok := f.Get()
		if !ok || item == nil {
			t.Fatalf("get %d failed", i)
		}
		if slot != uint32(i) {
			t.Fatalf("slot %d, want %d", slot, i)
		}
		*item = i + 1
		got = append(got, slot)
	}
	if _, _, ok := f.Get(); ok {
		t.Fatal("get past capacity")
	}
	if f.Available() != 0 || f.InUse() != 3 {
		t.Fatalf("exhausted: avail=%d inuse=%d", f.Available(), f.InUse())
	}

	if !f.Put(got[1]) {
		t.Fatal("put")
	}
	if f.Put(got[1]) {
		t.Fatal("double put")
	}
	if f.InUse() != 2 || f.Available() != 1 {
		t.Fatalf("after put: avail=%d inuse=%d", f.Available(), f.InUse())
	}

	item, slot, ok := f.Get()
	if !ok || slot != got[1] || *item != 0 {
		t.Fatalf("reget slot=%d val=%d ok=%v", slot, *item, ok)
	}

	if f.Put(99) {
		t.Fatal("put out of range")
	}
	if New[int](0) != nil || New[int](-1) != nil {
		t.Fatal("non-positive cap")
	}
}

func TestFreeListNoAlloc(t *testing.T) {
	f := New[int](8)
	if _, _, ok := f.Get(); !ok {
		t.Fatal("warmup get")
	}
	f.Put(0)
	allocs := testing.AllocsPerRun(200, func() {
		_, slot, ok := f.Get()
		if ok {
			f.Put(slot)
		}
	})
	if allocs != 0 {
		t.Fatalf("allocs/op = %v", allocs)
	}
}

func BenchmarkFreeListGetPut(b *testing.B) {
	f := New[int](1024)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, slot, ok := f.Get()
		if ok {
			f.Put(slot)
		}
	}
}
