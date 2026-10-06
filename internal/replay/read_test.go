package replay

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestReplayBytesMatchStreamFile(t *testing.T) {
	data, err := os.ReadFile("testdata/stream.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var s Session
	var steps int
	if err := s.Replay(data, func() { steps++ }); err != nil {
		t.Fatal(err)
	}
	if steps != 3 || s.Book.LastUpdate != 12 {
		t.Fatalf("steps %d last %d", steps, s.Book.LastUpdate)
	}
	if s.Book.Bids[100*Scale] != 2*Scale || s.Book.Asks[101*Scale] != 4*Scale {
		t.Fatalf("bids %+v asks %+v", s.Book.Bids, s.Book.Asks)
	}
}

func TestReplaySkipsBlankAndCommentLines(t *testing.T) {
	body := strings.Join([]string{
		"",
		"# ignored",
		`{"type":"snapshot","lastUpdateId":10,"bids":[["100","1"]]}`,
		"",
		`{"type":"diff","U":11,"u":11,"b":[["100","2"]]}`,
	}, "\n")
	var s Session
	if err := s.Replay([]byte(body), nil); err != nil {
		t.Fatal(err)
	}
	if s.Book.LastUpdate != 11 || s.Book.Bids[100*Scale] != 2*Scale {
		t.Fatalf("last %d bids %+v", s.Book.LastUpdate, s.Book.Bids)
	}
}

func TestReplayNamesTheBadLine(t *testing.T) {
	body := "# head\n{not json}\n"
	var s Session
	err := s.Replay([]byte(body), nil)
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("err %v", err)
	}
	if s.Book.synced {
		t.Fatal("bad line synced the book")
	}
}

func TestReplayReportsGapLine(t *testing.T) {
	body := strings.Join([]string{
		`{"type":"snapshot","lastUpdateId":10,"bids":[["100","1"]]}`,
		`{"type":"diff","U":13,"u":13,"b":[["100","2"]]}`,
	}, "\n")
	var s Session
	err := s.Replay([]byte(body), nil)
	if !errors.Is(err, ErrGap) || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("err %v", err)
	}
	if s.Book.LastUpdate != 10 || s.Book.Bids[100*Scale] != Scale {
		t.Fatalf("snapshot not kept: %+v", s.Book.Bids)
	}
}
