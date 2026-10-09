package engine

import "testing"

func checkTree(t *testing.T, tree *PriceTree) {
	t.Helper()
	if tree.Root != nil {
		if tree.Root.Parent != nil {
			t.Fatal("root parent")
		}
		if tree.Root.Color != colorBlack {
			t.Fatal("root color")
		}
	}
	var walk func(n *LimitLevel, lo, hi int64, hasLo, hasHi bool) int
	walk = func(n *LimitLevel, lo, hi int64, hasLo, hasHi bool) int {
		if n == nil {
			return 1
		}
		if n.Color != colorBlack && n.Color != colorRed {
			t.Fatalf("color %d at %d", n.Color, n.Price)
		}
		if n.Color == colorRed {
			if colorOf(n.Left) == colorRed || colorOf(n.Right) == colorRed {
				t.Fatalf("adjacent red at %d", n.Price)
			}
		}
		if hasLo && n.Price <= lo {
			t.Fatalf("bst lo %d at %d", lo, n.Price)
		}
		if hasHi && n.Price >= hi {
			t.Fatalf("bst hi %d at %d", hi, n.Price)
		}
		if n.Left != nil && n.Left.Parent != n {
			t.Fatalf("left parent at %d", n.Price)
		}
		if n.Right != nil && n.Right.Parent != n {
			t.Fatalf("right parent at %d", n.Price)
		}
		lh := walk(n.Left, lo, n.Price, hasLo, true)
		rh := walk(n.Right, n.Price, hi, true, hasHi)
		if lh != rh {
			t.Fatalf("black height %d vs %d at %d", lh, rh, n.Price)
		}
		if n.Color == colorBlack {
			return lh + 1
		}
		return lh
	}
	walk(tree.Root, 0, 0, false, false)
}

func inorder(n *LimitLevel, out []int64) []int64 {
	if n == nil {
		return out
	}
	out = inorder(n.Left, out)
	out = append(out, n.Price)
	return inorder(n.Right, out)
}

func TestPriceTreeMinMax(t *testing.T) {
	var tree PriceTree
	if tree.Min() != nil || tree.Max() != nil || tree.Find(1) != nil {
		t.Fatal("empty")
	}
	prices := []int64{5, 1, 4, 2, 3}
	nodes := make([]*LimitLevel, len(prices))
	for i, p := range prices {
		nodes[i] = &LimitLevel{Price: p}
		if !tree.Insert(nodes[i]) {
			t.Fatalf("insert %d", p)
		}
		checkTree(t, &tree)
	}
	if tree.Insert(&LimitLevel{Price: 4}) {
		t.Fatal("duplicate")
	}
	checkTree(t, &tree)
	if tree.Min().Price != 1 || tree.Max().Price != 5 {
		t.Fatalf("min %d max %d", tree.Min().Price, tree.Max().Price)
	}
	if tree.Find(3) != nodes[4] || tree.Find(9) != nil {
		t.Fatal("find")
	}
	got := inorder(tree.Root, nil)
	want := []int64{1, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatal(got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("inorder %v", got)
		}
	}

	for _, n := range nodes {
		tree.Delete(n)
		checkTree(t, &tree)
		if n.Left != nil || n.Right != nil || n.Parent != nil {
			t.Fatal("deleted links")
		}
	}
	if tree.Root != nil || tree.Min() != nil || tree.Max() != nil {
		t.Fatal("not empty")
	}
}

func TestPriceTreePermutations(t *testing.T) {
	const n = 5
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	var walk func(k int)
	walk = func(k int) {
		if k == n {
			var tree PriceTree
			nodes := make([]*LimitLevel, n)
			for i, id := range perm {
				nodes[i] = &LimitLevel{Price: int64(id + 1)}
				if !tree.Insert(nodes[i]) {
					t.Fatalf("insert %d", id)
				}
				checkTree(t, &tree)
			}
			if tree.Min().Price != 1 || tree.Max().Price != int64(n) {
				t.Fatalf("bounds min %d max %d perm %v", tree.Min().Price, tree.Max().Price, perm)
			}
			for i := n - 1; i >= 0; i-- {
				tree.Delete(nodes[i])
				checkTree(t, &tree)
			}
			if tree.Root != nil {
				t.Fatalf("leftover after %v", perm)
			}
			return
		}
		for i := k; i < n; i++ {
			perm[k], perm[i] = perm[i], perm[k]
			walk(k + 1)
			perm[k], perm[i] = perm[i], perm[k]
		}
	}
	walk(0)
}

func TestPriceTreeRandom(t *testing.T) {
	const n = 200
	rng := uint64(1)
	next := func() uint64 {
		rng ^= rng << 13
		rng ^= rng >> 7
		rng ^= rng << 17
		return rng
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := int(next() % uint64(i+1))
		order[i], order[j] = order[j], order[i]
	}

	var tree PriceTree
	nodes := make([]*LimitLevel, n)
	for _, id := range order {
		nodes[id] = &LimitLevel{Price: int64(id)}
		if !tree.Insert(nodes[id]) {
			t.Fatal("insert")
		}
	}
	checkTree(t, &tree)
	if tree.Min().Price != 0 || tree.Max().Price != int64(n-1) {
		t.Fatalf("min %d max %d", tree.Min().Price, tree.Max().Price)
	}

	for i := n - 1; i > 0; i-- {
		j := int(next() % uint64(i+1))
		order[i], order[j] = order[j], order[i]
	}
	for _, id := range order {
		tree.Delete(nodes[id])
		checkTree(t, &tree)
	}
	if tree.Root != nil {
		t.Fatal("not empty")
	}
}
