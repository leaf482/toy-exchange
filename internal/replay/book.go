package replay

import (
	"encoding/json"
	"errors"
	"sort"
)

// ErrGap means a diff does not continue the local update sequence.
var ErrGap = errors.New("depth sequence gap")

// ErrNotSynced means a diff arrived before a snapshot.
var ErrNotSynced = errors.New("diff before snapshot")

// Level is one price and its aggregated quantity, both in fixed point.
type Level struct {
	Price int64
	Qty   int64
}

// Book is a reconstructed L2 book. It is not the matching engine.
type Book struct {
	Bids       map[int64]int64
	Asks       map[int64]int64
	LastUpdate uint64
	fresh      bool
	synced     bool
}

// Message is one JSON object from a capture file.
// Snapshot objects carry lastUpdateId, bids, and asks.
// Diff objects carry U, u, and b/a or bids/asks.
type Message struct {
	Type         string     `json:"type"`
	Event        string     `json:"e"`
	LastUpdateID uint64     `json:"lastUpdateId"`
	U            uint64     `json:"U"`
	Final        uint64     `json:"u"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
	B            [][]string `json:"b"`
	A            [][]string `json:"a"`
}

// Decode parses one JSON line into a message.
func Decode(line []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(line, &m); err != nil {
		return Message{}, err
	}
	return m, nil
}

// Apply updates the book from one decoded message.
func (b *Book) Apply(m Message) error {
	if b.Bids == nil {
		b.Bids = make(map[int64]int64)
	}
	if b.Asks == nil {
		b.Asks = make(map[int64]int64)
	}
	if isSnapshot(m) {
		bids, err := parseSide(firstSide(m.Bids, nil))
		if err != nil {
			return err
		}
		asks, err := parseSide(firstSide(m.Asks, nil))
		if err != nil {
			return err
		}
		b.Bids = toMap(bids)
		b.Asks = toMap(asks)
		b.LastUpdate = m.LastUpdateID
		b.synced = true
		b.fresh = true
		return nil
	}
	return b.applyDiff(m)
}

func isSnapshot(m Message) bool {
	if m.Type == "snapshot" {
		return true
	}
	if m.Type == "diff" || m.Event == "depthUpdate" {
		return false
	}
	return m.LastUpdateID > 0 && m.U == 0 && m.Final == 0
}

func (b *Book) applyDiff(m Message) error {
	if !b.synced {
		return ErrNotSynced
	}
	if m.Final <= b.LastUpdate {
		return nil
	}
	if b.fresh {
		if m.U > b.LastUpdate+1 || m.Final < b.LastUpdate+1 {
			return ErrGap
		}
		b.fresh = false
	} else if m.U != b.LastUpdate+1 {
		return ErrGap
	}
	bids, err := parseSide(firstSide(m.B, m.Bids))
	if err != nil {
		return err
	}
	asks, err := parseSide(firstSide(m.A, m.Asks))
	if err != nil {
		return err
	}
	applyLevels(b.Bids, bids)
	applyLevels(b.Asks, asks)
	b.LastUpdate = m.Final
	return nil
}

func firstSide(a, b [][]string) [][]string {
	if len(a) > 0 {
		return a
	}
	return b
}

func parseSide(rows [][]string) ([]Level, error) {
	out := make([]Level, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			return nil, ErrDecimal
		}
		price, err := ParseFixed(row[0])
		if err != nil {
			return nil, err
		}
		qty, err := ParseFixed(row[1])
		if err != nil {
			return nil, err
		}
		out = append(out, Level{Price: price, Qty: qty})
	}
	return out, nil
}

func toMap(levels []Level) map[int64]int64 {
	m := make(map[int64]int64, len(levels))
	applyLevels(m, levels)
	return m
}

func applyLevels(dst map[int64]int64, levels []Level) {
	for _, lvl := range levels {
		if lvl.Qty == 0 {
			delete(dst, lvl.Price)
			continue
		}
		dst[lvl.Price] = lvl.Qty
	}
}

// BidsDesc returns bids from the best price downward.
func (b *Book) BidsDesc() []Level {
	if b == nil {
		return nil
	}
	return sortLevels(b.Bids, false)
}

// AsksAsc returns asks from the best price upward.
func (b *Book) AsksAsc() []Level {
	if b == nil {
		return nil
	}
	return sortLevels(b.Asks, true)
}

func sortLevels(m map[int64]int64, asc bool) []Level {
	out := make([]Level, 0, len(m))
	for price, qty := range m {
		out = append(out, Level{Price: price, Qty: qty})
	}
	sort.Slice(out, func(i, j int) bool {
		if asc {
			return out[i].Price < out[j].Price
		}
		return out[i].Price > out[j].Price
	})
	return out
}
