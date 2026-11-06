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
