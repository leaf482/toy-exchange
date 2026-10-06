package replay

// Session buffers depth diffs that arrive before the REST snapshot.
// After the snapshot, it drops events already covered by lastUpdateId and
// applies the rest in order. Diffs that arrive after the book is synced
// apply immediately.
type Session struct {
	Book Book
	buf  []Message
}

// Feed accepts one snapshot or diff.
// onUpdate runs after each book mutation. It may be nil.
// A diff received before the snapshot is stored and does not mutate the book.
func (s *Session) Feed(m Message, onUpdate func()) error {
	if s == nil {
		return ErrNotSynced
	}
	if onUpdate == nil {
		onUpdate = func() {}
	}
	if !isSnapshot(m) && !s.Book.synced {
		s.buf = append(s.buf, m)
		return nil
	}
	if isSnapshot(m) {
		if err := s.Book.Apply(m); err != nil {
			return err
		}
		onUpdate()
		return s.flush(onUpdate)
	}
	before := s.Book.LastUpdate
	if err := s.Book.Apply(m); err != nil {
		return err
	}
	if s.Book.LastUpdate != before {
		onUpdate()
	}
	return nil
}

func (s *Session) flush(onUpdate func()) error {
	pending := s.buf
	s.buf = nil
	for _, m := range pending {
		if m.Final <= s.Book.LastUpdate {
			continue
		}
		if err := s.Book.Apply(m); err != nil {
			return err
		}
		onUpdate()
	}
	return nil
}
