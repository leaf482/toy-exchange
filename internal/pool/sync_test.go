package pool

import (
	"sync"
	"testing"
)

// slot is the comparison value. The engine does not use sync.Pool.
type slot struct {
	n int
}

// TestSyncPoolAllocatesFromNew shows why the free list is the default.
// An empty sync.Pool calls New and allocates. FreeList.Get does not.
func TestSyncPoolAllocatesFromNew(t *testing.T) {
	var p sync.Pool
	p.New = func() any { return &slot{} }
	allocs := testing.AllocsPerRun(20, func() {
		p.Get()
	})
	if allocs == 0 {
		t.Fatal("empty sync.Pool Get allocated nothing")
	}
}

func BenchmarkSyncPoolGetPut(b *testing.B) {
	var p sync.Pool
	p.New = func() any { return &slot{} }
	p.Put(&slot{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := p.Get().(*slot)
		p.Put(v)
	}
}
