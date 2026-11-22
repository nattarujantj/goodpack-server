package models

import "testing"

func TestQuantityPerPackFrom(t *testing.T) {
	cases := []struct {
		qty, boxes, want int
		ok               bool
	}{
		{1000, 10, 100, true},
		{500, 1, 500, true},
		{1050, 11, 0, false},
		{100, 0, 0, false},
		{0, 5, 0, false},
		{-10, 2, 0, false},
	}
	for _, c := range cases {
		got, ok := QuantityPerPackFrom(c.qty, c.boxes)
		if got != c.want || ok != c.ok {
			t.Errorf("QuantityPerPackFrom(%d, %d) = %d, %v; want %d, %v", c.qty, c.boxes, got, ok, c.want, c.ok)
		}
	}
}

func TestWarehouseItemPerPack(t *testing.T) {
	if q, ok := (WarehouseItem{Quantity: 1050, Boxes: 11, QuantityPerPack: 100}).PerPack(); !ok || q != 100 {
		t.Errorf("explicit per pack: got %d, %v", q, ok)
	}
	if q, ok := (WarehouseItem{Quantity: 1000, Boxes: 10}).PerPack(); !ok || q != 100 {
		t.Errorf("derived per pack: got %d, %v", q, ok)
	}
	if _, ok := (WarehouseItem{Quantity: 1050, Boxes: 11}).PerPack(); ok {
		t.Errorf("uneven boxes should not derive per pack")
	}
}
