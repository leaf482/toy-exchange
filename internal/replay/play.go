package replay

// Play rebuilds a local book from a JSONL capture already in memory and
// returns the synthetic actions for that book.
// The matching engine is not called. The caller turns actions into an
// engine script only after Play returns.
func Play(data []byte) (Book, []Action, error) {
	var s Session
	synth := NewSynth()
	var actions []Action
	err := s.Replay(data, func() {
		actions = append(actions, synth.Push(s.Book.Bids, s.Book.Asks, int64(s.Book.LastUpdate))...)
	})
	return s.Book, actions, err
}
