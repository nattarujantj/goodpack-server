package models

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// ค่าคอมไม่คิด VAT แต่ต้องนับเป็นต้นทุนจริง
func TestImportVATExcludesCommission(t *testing.T) {
	req := &InternationalImportRequest{
		ImportType:   "LCL",
		UsdToThbRate: 30,
		PricePerCBM:  0,
		Items: []ImportItem{{
			ProductID:       "p1",
			UsdPricePerUnit: 10, // 300 บาท
			Quantity:        10,
			Commission:      20,
			CBMManual:       true,
		}},
	}
	imp := req.ToInternationalImport()
	item := imp.Items[0]

	if !almostEqual(item.CostPerUnitBeforeVAT, 320) {
		t.Fatalf("CostPerUnitBeforeVAT = %v, want 320", item.CostPerUnitBeforeVAT)
	}
	if !almostEqual(item.VATPerUnit, 21) { // 300 × 7%
		t.Fatalf("VATPerUnit = %v, want 21", item.VATPerUnit)
	}
	if !almostEqual(item.CostPerUnitAfterVAT, 341) {
		t.Fatalf("CostPerUnitAfterVAT = %v, want 341", item.CostPerUnitAfterVAT)
	}

	purchase := imp.ToPurchaseRequest(true).ToPurchase()
	pi := purchase.Items[0]
	if !almostEqual(pi.UnitPrice, 300) {
		t.Fatalf("purchase UnitPrice = %v, want 300", pi.UnitPrice)
	}
	if !almostEqual(purchase.TotalVAT, 210) {
		t.Fatalf("purchase TotalVAT = %v, want 210", purchase.TotalVAT)
	}
	if !almostEqual(pi.EffectiveUnitPrice(), 320) {
		t.Fatalf("EffectiveUnitPrice = %v, want 320", pi.EffectiveUnitPrice())
	}
}
