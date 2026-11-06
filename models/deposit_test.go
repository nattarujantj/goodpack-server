package models

import "testing"

func TestNormalizeDepositApplicationsMergesAndDropsEmpty(t *testing.T) {
	apps := NormalizeDepositApplications([]DepositApplication{
		{DepositID: "a", Amount: 10000},
		{DepositID: " ", Amount: 500},
		{DepositID: "b", Amount: 0},
		{DepositID: "a", Amount: 20000.004},
		{DepositID: "c", Amount: -5},
	})
	if len(apps) != 1 || apps[0].DepositID != "a" || apps[0].Amount != 30000 {
		t.Fatalf("unexpected result: %+v", apps)
	}
	if got := SumDepositApplications(apps); got != 30000 {
		t.Fatalf("sum = %v", got)
	}
}

// มัดจำ 100,000 หักขวด 300ml ไป 30,000 แล้วหักขวด 500ml อีก 70,000 ได้พอดี
func TestDepositUsedAcrossMultipleDocuments(t *testing.T) {
	d := &Deposit{Amount: 100000}
	d.ApplyUsages([]DepositUsage{{DocType: DepositDocPurchase, DocCode: "PO-1", Amount: 30000}})
	if d.UsedAmount != 30000 || d.RemainingAmount != 70000 {
		t.Fatalf("used=%v remaining=%v", d.UsedAmount, d.RemainingAmount)
	}
	if !d.CanCover(70000) {
		t.Fatal("remaining 70,000 should cover 70,000")
	}
	if d.CanCover(70000.01) {
		t.Fatal("remaining 70,000 should not cover 70,000.01")
	}
}

func TestDepositRequestValidate(t *testing.T) {
	r := DepositRequest{Type: "other", PartyID: "x", Amount: 1}
	if r.Validate() == nil {
		t.Fatal("invalid type should fail")
	}
	r = DepositRequest{Type: DepositTypeSupplier, PartyID: "x", Amount: 0}
	if r.Validate() == nil {
		t.Fatal("zero amount should fail")
	}
}

func TestPurchaseDepositIsSumOfApplications(t *testing.T) {
	req := PurchaseRequest{DepositApplications: []DepositApplication{{DepositID: "a", Amount: 30000}, {DepositID: "b", Amount: 5000}}}
	if p := req.ToPurchase(); p.Deposit != 35000 {
		t.Fatalf("deposit = %v", p.Deposit)
	}
}

func TestCalculateDepositVAT(t *testing.T) {
	b, v, total := CalculateDepositVAT(10000, true, "exclusive")
	if b != 10000 || v != 700 || total != 10700 {
		t.Fatalf("exclusive: %v %v %v", b, v, total)
	}
	b, v, total = CalculateDepositVAT(10700, true, "inclusive")
	if b != 10000 || v != 700 || total != 10700 {
		t.Fatalf("inclusive: %v %v %v", b, v, total)
	}
	b, v, total = CalculateDepositVAT(5000, false, "")
	if b != 5000 || v != 0 || total != 5000 {
		t.Fatalf("non-vat: %v %v %v", b, v, total)
	}
	d := &Deposit{IsVAT: true, Amount: 10700, AmountBeforeVAT: 10000}
	d.ApplyUsages([]DepositUsage{{Amount: 4000}})
	if d.DeductibleAmount != 10000 || d.RemainingAmount != 6000 {
		t.Fatalf("vat deposit balance: %+v", d)
	}
}

// ค่าสินค้า 100,000 หักมัดจำ (มีใบกำกับ) 10,000 ก่อน VAT → ฐาน 90,000 VAT 6,300 รวม 96,300
func TestSaleTotalsDeductVATDepositBeforeVAT(t *testing.T) {
	s := Sale{IsVAT: true, VatType: "exclusive", Items: []SaleItem{{TotalPrice: 100000}}, DepositBeforeVAT: 10000}
	tt := s.Totals()
	if tt.ItemsBeforeVAT != 100000 || tt.NetBeforeVAT != 90000 || tt.VAT != 6300 || tt.AfterVAT != 96300 {
		t.Fatalf("unexpected totals: %+v", tt)
	}
	s.DepositBeforeVAT = 0
	if tt := s.Totals(); tt.VAT != 7000 || tt.AfterVAT != 107000 {
		t.Fatalf("no deposit: %+v", tt)
	}
	s.VatType = "inclusive"
	s.Items = []SaleItem{{TotalPrice: 107000}}
	if tt := s.Totals(); tt.NetBeforeVAT != 100000 || tt.VAT != 7000 || tt.AfterVAT != 107000 {
		t.Fatalf("inclusive no deposit: %+v", tt)
	}
}

func TestPurchaseTotalsWithDeposits(t *testing.T) {
	req := PurchaseRequest{
		IsVAT: true, VATType: "exclusive",
		Items: []PurchaseItem{{TotalPrice: 100000}},
		DepositApplications: []DepositApplication{
			{DepositID: "vat", Amount: 30000, BeforeVAT: true},
			{DepositID: "nonvat", Amount: 5000},
		},
	}
	p := req.ToPurchase()
	if p.TotalAmount != 100000 || p.TotalVAT != 4900 || p.GrandTotal != 74900 {
		t.Fatalf("totals: amount=%v vat=%v grand=%v", p.TotalAmount, p.TotalVAT, p.GrandTotal)
	}
	if p.DepositBeforeVAT != 30000 || p.Deposit != 5000 {
		t.Fatalf("deposits: before=%v after=%v", p.DepositBeforeVAT, p.Deposit)
	}
}
