package index

import "testing"

func TestMapInsertLookupDelete(t *testing.T) {
	m := NewMap(4)
	if m == nil || m.Len() != 0 {
		t.Fatal("new")
	}
	if m.Insert(0, 1) {
		t.Fatal("id 0")
	}
	if !m.Insert(2, 0) {
		t.Fatal("insert slot 0")
	}
	if m.Insert(2, 3) {
		t.Fatal("duplicate")
	}
	slot, ok := m.Lookup(2)
	if !ok || slot != 0 {
		t.Fatalf("lookup %d %v", slot, ok)
	}
	m.Delete(2)
	if m.Has(2) || m.Len() != 0 {
		t.Fatal("deleted")
	}
	m.Delete(2)
	if NewMap(0) != nil {
		t.Fatal("hint")
	}
}
