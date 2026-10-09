// Package events is the external contract for match output.
// It does not know about trees or pools.
package events

// EventKind classifies one output record.
type EventKind uint8

const (
	EventAccepted        EventKind = 1
	EventPartiallyFilled EventKind = 2
	EventFilled          EventKind = 3
	EventCanceled        EventKind = 4
	EventRejected        EventKind = 5
	EventAmended         EventKind = 6
	EventTrade           EventKind = 7
)

// Reason explains a reject, cancel, or amend failure.
type Reason uint8

const (
	ReasonNone          Reason = 0
	ReasonBadQty        Reason = 1
	ReasonBadPrice      Reason = 2
	ReasonDuplicateID   Reason = 3
	ReasonNotFound      Reason = 4
	ReasonPoolExhausted Reason = 5
	ReasonIOCRemainder  Reason = 6
	ReasonBufferFull    Reason = 7
	ReasonBadType       Reason = 8
)

// Event is one trade or order-state record.
// Scalar fields only, so a pre-sized buffer can store it without allocation.
type Event struct {
	Kind      EventKind
	Reason    Reason
	OrderID   uint64
	MatchID   uint64
	TradeID   uint64
	Price     int64
	Qty       int64
	Leaves    int64
	Timestamp int64
}

// MaxEventsFor is the largest number of events one Submit, Cancel, or Amend
// can append when `resting` orders are live.
func MaxEventsFor(resting int32) int {
	return int(2*resting + 4)
}
