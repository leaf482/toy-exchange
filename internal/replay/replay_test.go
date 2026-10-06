package replay

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestParseFixed(t *testing.T) {
	n, err := ParseFixed("100.5")
	if err != nil || n != 100*Scale+Scale/2 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = ParseFixed("0.0024")
	if err != nil || n != 240000 {
		t.Fatalf("frac %d %v", n, err)
	}
	if _, err := ParseFixed("1.123456789"); err != ErrDecimal {
		t.Fatal(err)
	}
	if _, err := ParseFixed("nope"); err != ErrDecimal {
		t.Fatal(err)
	}
}

func TestCaptureReplay(t *testing.T) {
	f, err := os.Open("testdata/capture.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var book Book
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		msg, err := Decode([]byte(line))
		if err != nil {
			t.Fatal(err)
		}
		if err := book.Apply(msg); err != nil {
			t.Fatal(err)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	bids := book.BidsDesc()
	asks := book.AsksAsc()
	if len(bids) != 1 || bids[0].Price != 99*Scale || bids[0].Qty != 4*Scale {
		t.Fatalf("bids %+v", bids)
	}
	if len(asks) != 2 || asks[0].Price != 101*Scale || asks[0].Qty != 3*Scale {
		t.Fatalf("best ask %+v", asks)
	}
	if asks[1].Price != 102*Scale || asks[1].Qty != 1*Scale {
		t.Fatalf("second ask %+v", asks[1])
	}
	if book.LastUpdate != 12 {
		t.Fatalf("last %d", book.LastUpdate)
	}
}

func TestDiffGapAndStale(t *testing.T) {
	var book Book
	if err := book.Apply(Message{Type: "diff", U: 1, Final: 1}); err != ErrNotSynced {
		t.Fatal(err)
	}
	snap := Message{Type: "snapshot", LastUpdateID: 10, Bids: [][]string{{"100", "1"}}, Asks: [][]string{{"101", "1"}}}
	if err := book.Apply(snap); err != nil {
		t.Fatal(err)
	}
	stale := Message{Type: "diff", U: 8, Final: 9, B: [][]string{{"100", "9"}}}
	if err := book.Apply(stale); err != nil {
		t.Fatal(err)
	}
	if book.Bids[100*Scale] != Scale {
		t.Fatal("stale diff applied")
	}
	gap := Message{Type: "diff", U: 13, Final: 13, B: [][]string{{"100", "2"}}}
	if err := book.Apply(gap); err != ErrGap {
		t.Fatal(err)
	}
	ok := Message{Type: "diff", U: 11, Final: 11, B: [][]string{{"100", "0"}}}
	if err := book.Apply(ok); err != nil {
		t.Fatal(err)
	}
	if _, exists := book.Bids[100*Scale]; exists {
		t.Fatal("zero qty should delete")
	}
}
