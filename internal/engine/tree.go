package engine

const (
	colorBlack uint8 = 0
	colorRed   uint8 = 1
)

// PriceTree is an intrusive red-black tree of LimitLevel keyed by Price.
// Children that are absent are nil, and nil counts as black.
type PriceTree struct {
	Root *LimitLevel
}

func colorOf(n *LimitLevel) uint8 {
	if n == nil {
		return colorBlack
	}
	return n.Color
}

// Insert links level into the tree. A duplicate price is rejected.
// The level's tree links are overwritten. Queue fields are left alone.
func (t *PriceTree) Insert(z *LimitLevel) bool {
	if t == nil || z == nil {
		return false
	}
	var parent *LimitLevel
	cur := t.Root
	for cur != nil {
		parent = cur
		if z.Price == cur.Price {
			return false
		}
		if z.Price < cur.Price {
			cur = cur.Left
		} else {
			cur = cur.Right
		}
	}
	z.Left = nil
	z.Right = nil
	z.Parent = parent
	z.Color = colorRed
	if parent == nil {
		t.Root = z
	} else if z.Price < parent.Price {
		parent.Left = z
	} else {
		parent.Right = z
	}
	t.insertFixup(z)
	return true
}

func (t *PriceTree) insertFixup(z *LimitLevel) {
	for z.Parent != nil && z.Parent.Color == colorRed {
		p := z.Parent
		g := p.Parent
		if g == nil {
			break
		}
		if p == g.Left {
			y := g.Right
			if colorOf(y) == colorRed {
				p.Color = colorBlack
				y.Color = colorBlack
				g.Color = colorRed
				z = g
				continue
			}
			if z == p.Right {
				z = p
				t.rotateLeft(z)
				p = z.Parent
				g = nil
				if p != nil {
					g = p.Parent
				}
			}
			if p != nil {
				p.Color = colorBlack
			}
			if g != nil {
				g.Color = colorRed
				t.rotateRight(g)
			}
		} else {
			y := g.Left
			if colorOf(y) == colorRed {
				p.Color = colorBlack
				y.Color = colorBlack
				g.Color = colorRed
				z = g
				continue
			}
			if z == p.Left {
				z = p
				t.rotateRight(z)
				p = z.Parent
				g = nil
				if p != nil {
					g = p.Parent
				}
			}
			if p != nil {
				p.Color = colorBlack
			}
			if g != nil {
				g.Color = colorRed
				t.rotateLeft(g)
			}
		}
	}
	if t.Root != nil {
		t.Root.Color = colorBlack
	}
}

// Delete unlinks level. The level's tree links are cleared.
// Deleting a level that is not in this tree is undefined.
func (t *PriceTree) Delete(z *LimitLevel) {
	if t == nil || z == nil {
		return
	}
	y := z
	yOrig := y.Color
	var x *LimitLevel
	var xParent *LimitLevel

	if z.Left == nil {
		x = z.Right
		xParent = z.Parent
		t.transplant(z, z.Right)
	} else if z.Right == nil {
		x = z.Left
		xParent = z.Parent
		t.transplant(z, z.Left)
	} else {
		y = minNode(z.Right)
		yOrig = y.Color
		x = y.Right
		if y.Parent == z {
			xParent = y
		} else {
			xParent = y.Parent
			t.transplant(y, y.Right)
			y.Right = z.Right
			if y.Right != nil {
				y.Right.Parent = y
			}
		}
		t.transplant(z, y)
		y.Left = z.Left
		if y.Left != nil {
			y.Left.Parent = y
		}
		y.Color = z.Color
	}

	z.Left = nil
	z.Right = nil
	z.Parent = nil
	z.Color = colorBlack

	if yOrig == colorBlack {
		t.deleteFixup(x, xParent)
	}
}

// deleteFixup restores red-black balance after a black node is removed.
// x may be nil; xParent is the parent that would have owned x.
func (t *PriceTree) deleteFixup(x, parent *LimitLevel) {
	for x != t.Root && colorOf(x) == colorBlack {
		if parent == nil {
			break
		}
		if x == parent.Left {
			w := parent.Right
			if w != nil && w.Color == colorRed {
				w.Color = colorBlack
				parent.Color = colorRed
				t.rotateLeft(parent)
				w = parent.Right
			}
			if w == nil {
				x = parent
				parent = x.Parent
				continue
			}
			if colorOf(w.Left) == colorBlack && colorOf(w.Right) == colorBlack {
				w.Color = colorRed
				x = parent
				parent = x.Parent
				continue
			}
			if colorOf(w.Right) == colorBlack {
				if w.Left != nil {
					w.Left.Color = colorBlack
				}
				w.Color = colorRed
				t.rotateRight(w)
				w = parent.Right
			}
			if w != nil {
				w.Color = parent.Color
				if w.Right != nil {
					w.Right.Color = colorBlack
				}
			}
			parent.Color = colorBlack
			t.rotateLeft(parent)
			x = t.Root
			parent = nil
		} else {
			w := parent.Left
			if w != nil && w.Color == colorRed {
				w.Color = colorBlack
				parent.Color = colorRed
				t.rotateRight(parent)
				w = parent.Left
			}
			if w == nil {
				x = parent
				parent = x.Parent
				continue
			}
			if colorOf(w.Left) == colorBlack && colorOf(w.Right) == colorBlack {
				w.Color = colorRed
				x = parent
				parent = x.Parent
				continue
			}
			if colorOf(w.Left) == colorBlack {
				if w.Right != nil {
					w.Right.Color = colorBlack
				}
				w.Color = colorRed
				t.rotateLeft(w)
				w = parent.Left
			}
			if w != nil {
				w.Color = parent.Color
				if w.Left != nil {
					w.Left.Color = colorBlack
				}
			}
			parent.Color = colorBlack
			t.rotateRight(parent)
			x = t.Root
			parent = nil
		}
	}
	if x != nil {
		x.Color = colorBlack
	}
}

func (t *PriceTree) transplant(u, v *LimitLevel) {
	if u.Parent == nil {
		t.Root = v
	} else if u == u.Parent.Left {
		u.Parent.Left = v
	} else {
		u.Parent.Right = v
	}
	if v != nil {
		v.Parent = u.Parent
	}
}

func (t *PriceTree) rotateLeft(x *LimitLevel) {
	y := x.Right
	if y == nil {
		return
	}
	x.Right = y.Left
	if y.Left != nil {
		y.Left.Parent = x
	}
	y.Parent = x.Parent
	if x.Parent == nil {
		t.Root = y
	} else if x == x.Parent.Left {
		x.Parent.Left = y
	} else {
		x.Parent.Right = y
	}
	y.Left = x
	x.Parent = y
}

func (t *PriceTree) rotateRight(x *LimitLevel) {
	y := x.Left
	if y == nil {
		return
	}
	x.Left = y.Right
	if y.Right != nil {
		y.Right.Parent = x
	}
	y.Parent = x.Parent
	if x.Parent == nil {
		t.Root = y
	} else if x == x.Parent.Left {
		x.Parent.Left = y
	} else {
		x.Parent.Right = y
	}
	y.Right = x
	x.Parent = y
}

// Min returns the lowest price, or nil when the tree is empty.
func (t *PriceTree) Min() *LimitLevel {
	if t == nil {
		return nil
	}
	return minNode(t.Root)
}

// Max returns the highest price, or nil when the tree is empty.
func (t *PriceTree) Max() *LimitLevel {
	if t == nil {
		return nil
	}
	n := t.Root
	if n == nil {
		return nil
	}
	for n.Right != nil {
		n = n.Right
	}
	return n
}

func minNode(n *LimitLevel) *LimitLevel {
	if n == nil {
		return nil
	}
	for n.Left != nil {
		n = n.Left
	}
	return n
}

// Prev returns the next-lower price, or nil.
func (t *PriceTree) Prev(n *LimitLevel) *LimitLevel {
	if n == nil {
		return nil
	}
	if n.Left != nil {
		m := n.Left
		for m.Right != nil {
			m = m.Right
		}
		return m
	}
	p := n.Parent
	for p != nil && n == p.Left {
		n = p
		p = p.Parent
	}
	return p
}

// Next returns the next-higher price, or nil.
func (t *PriceTree) Next(n *LimitLevel) *LimitLevel {
	if n == nil {
		return nil
	}
	if n.Right != nil {
		return minNode(n.Right)
	}
	p := n.Parent
	for p != nil && n == p.Right {
		n = p
		p = p.Parent
	}
	return p
}

// Find returns the level at price, or nil.
func (t *PriceTree) Find(price int64) *LimitLevel {
	if t == nil {
		return nil
	}
	n := t.Root
	for n != nil {
		if price == n.Price {
			return n
		}
		if price < n.Price {
			n = n.Left
		} else {
			n = n.Right
		}
	}
	return nil
}
