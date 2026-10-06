package replay

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestSessionBuffersUntilSnapshot(t *testing.T) {
	var s Session
	early := Message{Type: "diff", U: 8, Final: 9, B: [][]string{{"100", "9"}}}
	if err := s.Feed(early, nil); err != nil {
		t.Fatal(err)
	}
	if s.Book.synced || len(s.Book.Bids) != 0 {
		t.Fatal("diff applied before snapshot")
	}

	var steps int
	err := s.Feed(Message{
		Type: "snapshot", LastUpdateID: 10,
		Bids: [][]string{{"100", "1"}},
		Asks: [][]string{{"101", "1"}},
	}, func() { steps++ })
	if err != nil {
		t.Fatal(err)
	}
	if steps != 1 {
		t.Fatalf("steps %d", steps)
	}
	if s.Book.Bids[100*Scale] != Scale || s.Book.LastUpdate != 10 {
		t.Fatalf("stale diff was applied: %+v last %d", s.Book.Bids, s.Book.LastUpdate)
	}

	err = s.Feed(Message{Type: "diff", U: 11, Final: 11, B: [][]string{{"100", "2"}}}, func() { steps++ })
	if err != nil {
		t.Fatal(err)
	}
	err = s.Feed(Message{Type: "diff", U: 12, Final: 12, A: [][]string{{"101", "4"}}}, func() { steps++ })
	if err != nil {
		t.Fatal(err)
	}
	if steps != 3 || s.Book.LastUpdate != 12 {
		t.Fatalf("steps %d last %d", steps, s.Book.LastUpdate)
	}
	if s.Book.Bids[100*Scale] != 2*Scale || s.Book.Asks[101*Scale] != 4*Scale {
		t.Fatalf("bids %+v asks %+v", s.Book.Bids, s.Book.Asks)
	}
}

func TestSessionFlushesBridgeEvent(t *testing.T) {
	var s Session
	if err := s.Feed(Message{Type: "diff", U: 8, Final: 9, B: [][]string{{"100", "9"}}}, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Feed(Message{Type: "diff", U: 11, Final: 11, B: [][]string{{"100", "2"}}}, nil); err != nil {
		t.Fatal(err)
	}
	var steps int
	err := s.Feed(Message{
		Type: "snapshot", LastUpdateID: 10,
		Bids: [][]string{{"100", "1"}},
		Asks: [][]string{{"101", "1"}},
	}, func() { steps++ })
	if err != nil {
		t.Fatal(err)
	}
	if steps != 2 || s.Book.LastUpdate != 11 || s.Book.Bids[100*Scale] != 2*Scale {
		t.Fatalf("steps %d last %d bids %+v", steps, s.Book.LastUpdate, s.Book.Bids)
	}
}

func TestStreamFile(t *testing.T) {
	f, err := os.Open("testdata/stream.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var s Session
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		msg, err := Decode([]byte(line))
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Feed(msg, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if s.Book.LastUpdate != 12 {
		t.Fatalf("last %d", s.Book.LastUpdate)
	}
	if s.Book.Bids[100*Scale] != 2*Scale || s.Book.Asks[101*Scale] != 4*Scale {
		t.Fatalf("bids %+v asks %+v", s.Book.Bids, s.Book.Asks)
	}
}

func TestSessionGapInBuffer(t *testing.T) {
	var s Session
	if err := s.Feed(Message{Type: "diff", U: 13, Final: 13, B: [][]string{{"100", "2"}}}, nil); err != nil {
		t.Fatal(err)
	}
	err := s.Feed(Message{
		Type: "snapshot", LastUpdateID: 10,
		Bids: [][]string{{"100", "1"}},
	}, nil)
	if err != ErrGap {
		t.Fatalf("err %v", err)
	}
	if s.Book.LastUpdate != 10 || s.Book.Bids[100*Scale] != Scale {
		t.Fatalf("snapshot not kept: %+v", s.Book.Bids)
	}
}
