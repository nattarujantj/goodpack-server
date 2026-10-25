package models

import "testing"

// VAT ของค่าใช้จ่ายตู้เป็นภาษีซื้อ ไม่นับเป็นค่าตู้ที่เฉลี่ยเข้าต้นทุนสินค้า
func TestFCLCostLineVATIsNotContainerCost(t *testing.T) {
	r := &FCLShipmentRequest{
		UsdToThbRate: 35,
		CostLines: []FCLCostLine{
			{Name: "ค่าระวางเรือ", Currency: "USD", Amount: 1000},
			{Name: "THC", Currency: "THB", Amount: 6500, VAT: 455, Vendor: "MPJ", TaxInvoiceNo: "IV1"},
			{Name: "ค่าเคลียร์", Currency: "THB", Amount: 3000, VAT: 210},
		},
	}
	s := r.ToFCLShipment()
	if !almostEqual(s.TotalCostThb, 35000+6500+3000) {
		t.Fatalf("TotalCostThb = %v, want 44500", s.TotalCostThb)
	}
	if !almostEqual(s.TotalVAT, 665) {
		t.Fatalf("TotalVAT = %v, want 665", s.TotalVAT)
	}
	if s.CostLines[1].Vendor != "MPJ" || s.CostLines[1].TaxInvoiceNo != "IV1" {
		t.Fatalf("line details not kept: %+v", s.CostLines[1])
	}
}
