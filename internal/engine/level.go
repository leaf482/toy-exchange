package engine

// PushBack appends an order at the tail. FIFO pops from Head.
// An order that already belongs to a level is left where it is.
func (l *LimitLevel) PushBack(o *Order) {
	if l == nil || o == nil || o.Level != nil {
		return
	}
	o.Level = l
	o.Next = nil
	o.Prev = l.Tail
	if l.Tail != nil {
		l.Tail.Next = o
	} else {
		l.Head = o
	}
	l.Tail = o
	l.Count++
	l.TotalQty += o.Leaves
}

// Unlink removes an order from this level in O(1).
// An order that is not on this level is left unchanged.
func (l *LimitLevel) Unlink(o *Order) {
	if l == nil || o == nil || o.Level != l {
		return
	}
	if o.Prev != nil {
		o.Prev.Next = o.Next
	} else {
		l.Head = o.Next
	}
	if o.Next != nil {
		o.Next.Prev = o.Prev
	} else {
		l.Tail = o.Prev
	}
	l.Count--
	l.TotalQty -= o.Leaves
	o.Prev = nil
	o.Next = nil
	o.Level = nil
}

// PopFront detaches Head and returns it. An empty level returns nil.
func (l *LimitLevel) PopFront() *Order {
	if l == nil || l.Head == nil {
		return nil
	}
	o := l.Head
	l.Unlink(o)
	return o
}
