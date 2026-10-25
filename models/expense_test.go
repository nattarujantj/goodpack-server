package models

import "testing"

func TestExpenseVAT(t *testing.T) {
	e := &Expense{Amount: 1070, VAT: 70}
	if !almostEqual(e.AmountBeforeVAT(), 1000) {
		t.Fatalf("AmountBeforeVAT = %v, want 1000", e.AmountBeforeVAT())
	}

	cases := []struct {
		name string
		req  ExpenseRequest
		ok   bool
	}{
		{"no VAT", ExpenseRequest{Category: "ค่าขนส่ง", Amount: 500}, true},
		{"with VAT", ExpenseRequest{Category: "ค่าขนส่ง", Amount: 1070, VAT: 70}, true},
		{"negative VAT", ExpenseRequest{Category: "ค่าขนส่ง", Amount: 1070, VAT: -1}, false},
		{"VAT not less than amount", ExpenseRequest{Category: "ค่าขนส่ง", Amount: 100, VAT: 100}, false},
		{"missing category", ExpenseRequest{Amount: 100}, false},
	}
	for _, c := range cases {
		if err := c.req.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: Validate() = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}
