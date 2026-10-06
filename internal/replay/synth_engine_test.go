package replay_test

import (
	"os"
	"testing"

	"github.com/leaf482/toy-exchange/internal/engine"
	"github.com/leaf482/toy-exchange/internal/replay"
	"github.com/leaf482/toy-exchange/pkg/events"
)

func TestCaptureScriptMatchesEngineBook(t *testing.T) {
	matchFile(t, "testdata/capture.jsonl")
}

func TestStreamScriptMatchesEngineBook(t *testing.T) {
	matchFile(t, "testdata/stream.jsonl")
}

func matchFile(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	book, actions, err := replay.Play(data)
	if err != nil {
		t.Fatal(err)
	}
	ob, err := engine.New(32, 32, 32)
	if err != nil {
		t.Fatal(err)
	}
	applyOps(t, ob, replay.EngineOps(actions))
	if err := ob.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	assertDepth(t, ob, engine.SideBuy, book.BidsDesc())
	assertDepth(t, ob, engine.SideSell, book.AsksAsc())
}

func applyOps(t *testing.T, ob *engine.OrderBook, ops []engine.Op) {
	t.Helper()
	dst := make([]events.Event, 0, 64)
	for _, op := range ops {
		var err error
		dst, err = ob.Apply(op, dst[:0])
		if err != nil {
			t.Fatal(err)
		}
	}
}

func assertDepth(t *testing.T, ob *engine.OrderBook, side engine.Side, want []replay.Level) {
	t.Helper()
	got := ob.Depth(side, len(want)+1, make([]engine.LevelView, len(want)+1))
	if len(got) != len(want) {
		t.Fatalf("side %d depth %d want %d", side, len(got), len(want))
	}
	for i := range want {
		if got[i].Price != want[i].Price || got[i].Qty != want[i].Qty || got[i].Count != 1 {
			t.Fatalf("level %d got %+v want %+v", i, got[i], want[i])
		}
	}
}
