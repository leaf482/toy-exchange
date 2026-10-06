package replay

import (
	"errors"
	"os"
	"testing"
)

func TestPlayStream(t *testing.T) {
	data, err := os.ReadFile("testdata/stream.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	book, actions, err := Play(data)
	if err != nil {
		t.Fatal(err)
	}
	if book.LastUpdate != 12 || book.Bids[100*Scale] != 2*Scale || book.Asks[101*Scale] != 4*Scale {
		t.Fatalf("book %+v", book)
	}
	if len(actions) == 0 {
		t.Fatal("no actions")
	}
	ops := EngineOps(actions)
	if len(ops) != len(actions) {
		t.Fatalf("ops %d actions %d", len(ops), len(actions))
	}
}

func TestPlayCapture(t *testing.T) {
	data, err := os.ReadFile("testdata/capture.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	book, actions, err := Play(data)
	if err != nil {
		t.Fatal(err)
	}
	if book.LastUpdate != 12 || book.Bids[99*Scale] != 4*Scale {
		t.Fatalf("book %+v", book)
	}
	if book.Asks[101*Scale] != 3*Scale || book.Asks[102*Scale] != Scale {
		t.Fatalf("asks %+v", book.Asks)
	}
	if len(actions) == 0 {
		t.Fatal("no actions")
	}
}

func TestPlayReportsGap(t *testing.T) {
	body := []byte("{\"type\":\"snapshot\",\"lastUpdateId\":10,\"bids\":[[\"100\",\"1\"]]}\n{\"type\":\"diff\",\"U\":13,\"u\":13,\"b\":[[\"100\",\"2\"]]}\n")
	book, _, err := Play(body)
	if !errors.Is(err, ErrGap) {
		t.Fatalf("err %v", err)
	}
	if book.LastUpdate != 10 || book.Bids[100*Scale] != Scale {
		t.Fatalf("snapshot not kept: %+v", book.Bids)
	}
}
