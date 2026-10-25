package models

import (
	"math"
	"testing"
	"time"
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

	purchase := imp.ToPurchaseRequest(ImportVATModeVAT).ToPurchase()
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

// ตัวเลขจากใบขนจริง A002-06910-04644: CIF 316,678.77 / F 125,022.53 / VAT 22,167.00
func TestCustomsSplitVAT(t *testing.T) {
	c := &ImportCustoms{CIFValue: 316678.77, FreightValue: 125022.53, VAT: 22167}
	goods, freight := c.SplitVAT()
	if !almostEqual(freight, 8751.37) || !almostEqual(goods, 13415.63) {
		t.Fatalf("SplitVAT = %v / %v, want 13415.63 / 8751.37", goods, freight)
	}
}

// FCL ที่กรอกใบขนแล้ว: VAT ค่าของเฉลี่ยตามมูลค่า, VAT ค่าเฟรทเฉลี่ยตาม CBM ภายใน imp
func TestImportCustomsVAT(t *testing.T) {
	declDate := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	imp := &InternationalImport{
		ImportType:   "FCL",
		UsdToThbRate: 35,
		// VAT 1,000 แบ่งเป็นค่าเฟรท 1,000 × 250 / 1,000 = 250, ค่าของ 750
		ImportDate: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
		Customs:    &ImportCustoms{DeclarationNo: "A001", DeclarationDate: &declDate, CIFValue: 1000, FreightValue: 250, VAT: 1000},
		Items: []ImportItem{
			// ค่าของ 35 × 100 = 3,500 (มูลค่า 50%), CBM 1 (25%)
			{ProductID: "a", UsdPricePerUnit: 1, Quantity: 100, CBM: 1, CBMManual: true},
			// ค่าของ 70 × 50 = 3,500 (มูลค่า 50%), CBM 3 (75%)
			{ProductID: "b", UsdPricePerUnit: 2, Quantity: 50, CBM: 3, CBMManual: true},
		},
	}
	// ตู้: ค่าตู้ 8,000 / CBM รวมทั้งตู้ 8 (มี imp อื่นด้วย)
	imp.RecalculateForFCLContainer(8000, 8)

	if imp.VATSource != VATSourceCustoms {
		t.Fatalf("VATSource = %q, want customs", imp.VATSource)
	}
	a, b := imp.Items[0], imp.Items[1]
	if !almostEqual(a.ShippingCostPerUnit, 10) || !almostEqual(b.ShippingCostPerUnit, 60) {
		t.Fatalf("shipping = %v / %v, want 10 / 60", a.ShippingCostPerUnit, b.ShippingCostPerUnit)
	}
	if !almostEqual(a.GoodsVATPerUnit, 3.75) || !almostEqual(b.GoodsVATPerUnit, 7.5) {
		t.Fatalf("goods VAT = %v / %v, want 3.75 / 7.5", a.GoodsVATPerUnit, b.GoodsVATPerUnit)
	}
	if !almostEqual(a.FreightVATPerUnit, 0.625) || !almostEqual(b.FreightVATPerUnit, 3.75) {
		t.Fatalf("freight VAT = %v / %v, want 0.625 / 3.75", a.FreightVATPerUnit, b.FreightVATPerUnit)
	}
	if !almostEqual(a.CostPerUnitAfterVAT, 35+10+3.75+0.625) {
		t.Fatalf("CostPerUnitAfterVAT = %v", a.CostPerUnitAfterVAT)
	}
	if !almostEqual(imp.TotalGoodsVAT, 750) || !almostEqual(imp.TotalFreightVAT, 250) {
		t.Fatalf("totals = %v / %v, want 750 / 250", imp.TotalGoodsVAT, imp.TotalFreightVAT)
	}
	if !imp.CanCreateImportPurchase() {
		t.Fatal("CanCreateImportPurchase = false, want true")
	}

	purchase := imp.ToPurchaseRequest(ImportVATModeImport).ToPurchase()
	if !purchase.IsVAT || purchase.VATType != VATTypeImport {
		t.Fatalf("purchase isVAT=%v vatType=%q, want true/import", purchase.IsVAT, purchase.VATType)
	}
	if !almostEqual(purchase.TotalAmount, 3500+1000+3500+3000) || !almostEqual(purchase.TotalVAT, 1000) {
		t.Fatalf("purchase totals = %v / %v, want 11000 / 1000", purchase.TotalAmount, purchase.TotalVAT)
	}
	if purchase.ImportCustoms == nil || purchase.ImportCustoms.GoodsVAT != 750 || purchase.ImportCustoms.FreightVAT != 250 {
		t.Fatalf("ImportCustoms = %+v, want goods 750 freight 250", purchase.ImportCustoms)
	}
	if purchase.InvoiceNumber == nil || *purchase.InvoiceNumber != "A001" {
		t.Fatalf("InvoiceNumber = %v, want A001", purchase.InvoiceNumber)
	}
	if !purchase.PurchaseDate.Equal(declDate) {
		t.Fatalf("PurchaseDate = %v, want declaration date %v", purchase.PurchaseDate, declDate)
	}
	if !almostEqual(purchase.Items[0].ImportCostPerUnit(), a.CostPerUnitAfterVAT) {
		t.Fatalf("ImportCostPerUnit = %v, want %v", purchase.Items[0].ImportCostPerUnit(), a.CostPerUnitAfterVAT)
	}
}

// VAT ที่เฉลี่ยแล้วต้องรวมกันได้ตรงกับใบขนทุกสตางค์
func TestImportCustomsVATRoundingMatchesDeclaration(t *testing.T) {
	imp := &InternationalImport{
		ImportType:   "FCL",
		UsdToThbRate: 33.3333,
		Customs:      &ImportCustoms{CIFValue: 316678.77, FreightValue: 125022.53, ImportDuty: 333.33, VAT: 22167},
		Items: []ImportItem{
			{ProductID: "a", UsdPricePerUnit: 1.11, Quantity: 7, CBM: 1, CBMManual: true},
			{ProductID: "b", UsdPricePerUnit: 2.22, Quantity: 13, CBM: 1, CBMManual: true},
			{ProductID: "c", UsdPricePerUnit: 3.33, Quantity: 17, CBM: 1, CBMManual: true},
		},
	}
	imp.RecalculateForFCLContainer(10000, 9)
	purchase := imp.ToPurchaseRequest(ImportVATModeImport).ToPurchase()
	if !almostEqual(purchase.TotalVAT, 22167) {
		t.Fatalf("TotalVAT = %v, want 22167", purchase.TotalVAT)
	}
	if !almostEqual(math.Round(imp.TotalImportDuty*100)/100, 333.33) {
		t.Fatalf("TotalImportDuty = %v, want 333.33", imp.TotalImportDuty)
	}
}

// ยังไม่กรอกใบขน: ประมาณ 7% เหมือนเดิม และสร้างใบซื้อแบบนำเข้าไม่ได้
func TestImportWithoutCustomsIsEstimated(t *testing.T) {
	imp := &InternationalImport{
		ImportType:   "FCL",
		UsdToThbRate: 35,
		Items:        []ImportItem{{ProductID: "p1", UsdPricePerUnit: 1, Quantity: 1000, CBM: 10, CBMManual: true}},
	}
	imp.RecalculateForFCLContainer(50000, 20)
	if imp.VATSource != VATSourceEstimated || imp.CanCreateImportPurchase() {
		t.Fatalf("VATSource = %q, want estimated and no import purchase", imp.VATSource)
	}
	if !almostEqual(imp.Items[0].VATPerUnit, 60*0.07) {
		t.Fatalf("VATPerUnit = %v, want 4.2", imp.Items[0].VATPerUnit)
	}
}
