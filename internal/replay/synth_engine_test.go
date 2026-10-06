package replay_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/leaf482/toy-exchange/internal/engine"
	"github.com/leaf482/toy-exchange/internal/replay"
	"github.com/leaf482/toy-exchange/pkg/events"
)

func TestCaptureScriptMatchesEngineBook(t *testing.T) {
	f, err := os.Open("testdata/capture.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var book replay.Book
	synth := replay.NewSynth()
	var actions []replay.Action
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		msg, err := replay.Decode([]byte(line))
		if err != nil {
			t.Fatal(err)
		}
		if err := book.Apply(msg); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, synth.Push(book.Bids, book.Asks, int64(book.LastUpdate))...)
	}
	if err := sc.Err(); err != nil {
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

func TestStreamScriptMatchesEngineBook(t *testing.T) {
	data, err := os.ReadFile("testdata/stream.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var session replay.Session
	synth := replay.NewSynth()
	var actions []replay.Action
	if err := session.Replay(data, func() {
		actions = append(actions, synth.Push(session.Book.Bids, session.Book.Asks, int64(session.Book.LastUpdate))...)
	}); err != nil {
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
	assertDepth(t, ob, engine.SideBuy, session.Book.BidsDesc())
	assertDepth(t, ob, engine.SideSell, session.Book.AsksAsc())
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
