package replay

import "testing"

func TestSynthRules(t *testing.T) {
	s := NewSynth()
	first := s.Push(map[int64]int64{100: 5, 90: 1}, map[int64]int64{110: 2}, 1)
	want := []Action{
		{Kind: ActionLimit, ID: 1, Side: SideBuy, Price: 100, Qty: 5, Time: 1},
		{Kind: ActionLimit, ID: 2, Side: SideBuy, Price: 90, Qty: 1, Time: 1},
		{Kind: ActionLimit, ID: 3, Side: SideSell, Price: 110, Qty: 2, Time: 1},
	}
	assertActions(t, first, want)

	if again := s.Push(map[int64]int64{100: 5, 90: 1}, map[int64]int64{110: 2}, 2); len(again) != 0 {
		t.Fatalf("unchanged book emitted %#v", again)
	}

	second := s.Push(map[int64]int64{100: 5, 95: 4}, map[int64]int64{110: 4}, 3)
	want = []Action{
		{Kind: ActionLimit, ID: 4, Side: SideBuy, Price: 95, Qty: 4, Time: 3},
		{Kind: ActionCancel, CancelID: 2, Time: 3},
		{Kind: ActionCancel, CancelID: 3, Time: 3},
		{Kind: ActionLimit, ID: 5, Side: SideSell, Price: 110, Qty: 4, Time: 3},
	}
	assertActions(t, second, want)

	third := s.Push(map[int64]int64{100: 3, 95: 4}, nil, 4)
	want = []Action{
		{Kind: ActionCancel, CancelID: 1, Time: 4},
		{Kind: ActionLimit, ID: 6, Side: SideBuy, Price: 100, Qty: 3, Time: 4},
		{Kind: ActionCancel, CancelID: 5, Time: 4},
	}
	assertActions(t, third, want)
}

func assertActions(t *testing.T, got, want []Action) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len %d want %d\n%#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("action %d\n got %#v\nwant %#v", i, got[i], want[i])
		}
	}
}
