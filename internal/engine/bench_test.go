package engine

import (
	"math/bits"
	"os"
	"testing"
	"time"

	"github.com/leaf482/toy-exchange/pkg/events"
)

func TestScriptPrefixMatchesReference(t *testing.T) {
	const n = 1000
	script := BuildScript(n, 1)
	b := newBook(t, n, 64, n)
	ref := newRef()
	dstB := evbuf(events.MaxEventsFor(int32(n)))
	dstR := evbuf(events.MaxEventsFor(int32(n)))
	var logB, logR string
	for _, op := range script {
		var errB, errR error
		dstB, errB = b.Apply(op, dstB[:0])
		dstR, errR = applyRef(ref, op, dstR[:0])
		if (errB == nil) != (errR == nil) {
			t.Fatalf("err %v %v", errB, errR)
		}
		logB += formatEvents(dstB)
		logR += formatEvents(dstR)
	}
	if logB != logR {
		t.Fatal("engine and reference diverged on the seed-1 prefix")
	}
	if err := b.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	const path = "../../test/fixtures/script_1000.out.txt"
	if os.Getenv("WRITE_FIXTURE") == "1" {
		if err := os.WriteFile(path, []byte(logB), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != logB {
		t.Fatal("script_1000.out.txt does not match seed 1")
	}
}

func applyRef(r *refBook, op Op, dst []events.Event) ([]events.Event, error) {
	if op.Kind == OpCancel {
		return r.Cancel(op.CancelID, dst)
	}
	return r.Submit(op.In, dst)
}

func TestMillionScript(t *testing.T) {
	const n = 1_000_000
	script := BuildScript(n, 1)
	book, err := New(n, 64, n)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]events.Event, 0, events.MaxEventsFor(int32(n)))
	for _, op := range script {
		dst, err = book.Apply(op, dst[:0])
		if err != nil && err != ErrNotFound {
			t.Fatal(err)
		}
	}
	if book.NextTradeID <= 1 {
		t.Fatal("script produced no trades")
	}
	if err := book.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestP99Histogram(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	const n = 1_000_000
	script := BuildScript(n, 1)
	book, err := New(n, 64, n)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]events.Event, 0, events.MaxEventsFor(int32(n)))
	var h hist
	for _, op := range script {
		dst = dst[:0]
		start := time.Now()
		dst, err = book.Apply(op, dst)
		elapsed := time.Since(start).Nanoseconds()
		if err != nil && err != ErrNotFound {
			t.Fatal(err)
		}
		h.Add(elapsed)
	}
	t.Logf("P99 bucket upper edge %dns over %d ops buckets %v", h.P99(), n, h.b)
}

// hist counts nanoseconds into power-of-two buckets.
// Bucket i covers (2^(i-1), 2^i] for i >= 1. Bucket 0 covers values below 2.
// The last bucket is overflow above 2^20.
type hist struct {
	b [22]uint64
}

func (h *hist) Add(ns int64) {
	if ns < 2 {
		h.b[0]++
		return
	}
	i := bits.Len64(uint64(ns)) - 1
	if i > 20 {
		h.b[21]++
		return
	}
	h.b[i]++
}

func (h *hist) P99() int64 {
	var total uint64
	for _, c := range h.b {
		total += c
	}
	if total == 0 {
		return 0
	}
	need := (total*99 + 99) / 100
	var cum uint64
	for i, c := range h.b {
		cum += c
		if cum >= need {
			if i >= 21 {
				return 1 << 21
			}
			return 1 << i
		}
	}
	return 1 << 21
}

func TestHotPathNoAlloc(t *testing.T) {
	b := newBook(t, 128, 32, 128)
	dst := make([]events.Event, 0, events.MaxEventsFor(128))
	if _, err := b.Submit(OrderInput{ID: 1, Side: SideBuy, Type: TypeLimit, Price: 10, Qty: 1, Timestamp: 1}, dst[:0]); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Cancel(1, dst[:0]); err != nil {
		t.Fatal(err)
	}
	var id uint64 = 2
	allocs := testing.AllocsPerRun(100, func() {
		dst = dst[:0]
		if _, err := b.Submit(OrderInput{ID: id, Side: SideBuy, Type: TypeLimit, Price: 10, Qty: 1, Timestamp: int64(id)}, dst); err != nil {
			t.Fatalf("submit %v", err)
		}
		dst = dst[:0]
		if _, err := b.Cancel(id, dst); err != nil {
			t.Fatalf("cancel %v", err)
		}
		id++
	})
	if allocs != 0 {
		t.Fatalf("allocs/op = %v", allocs)
	}
}

func BenchmarkMillion(b *testing.B) {
	const n = 1_000_000
	script := BuildScript(n, 1)
	dst := make([]events.Event, 0, events.MaxEventsFor(int32(n)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		book, err := New(n, 64, n)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		for _, op := range script {
			dst = dst[:0]
			var err error
			dst, err = book.Apply(op, dst)
			if err != nil && err != ErrNotFound {
				b.Fatal(err)
			}
		}
	}
}
