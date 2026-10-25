package models

import (
	"math"
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InternationalImport struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ImportCode          string             `bson:"importCode" json:"importCode"`
	ImportDate          time.Time          `bson:"importDate" json:"importDate"`
	ImportType          string             `bson:"importType" json:"importType"` // "LCL" or "FCL"
	SupplierID          string             `bson:"supplierId" json:"supplierId"`
	SupplierName        string             `bson:"supplierName" json:"supplierName"`
	ShippingCompanyID   string             `bson:"shippingCompanyId" json:"shippingCompanyId"`
	ShippingCompanyName string             `bson:"shippingCompanyName" json:"shippingCompanyName"`
	UsdToThbRate        float64            `bson:"usdToThbRate" json:"usdToThbRate"`
	// LCL specific
	PricePerCBM float64 `bson:"pricePerCBM" json:"pricePerCBM"`
	// FCL specific
	FCLCostDetails []FCLCostDetail `bson:"fclCostDetails" json:"fclCostDetails"`
	TotalFCLCost   float64         `bson:"totalFCLCost" json:"totalFCLCost"`
	// FCLShipmentID links this import (one factory / imp id) to a shared container.
	// When set, shipping cost is derived from the container instead of the fields above.
	FCLShipmentID *string `bson:"fclShipmentId,omitempty" json:"fclShipmentId,omitempty"`
	// Customs คือข้อมูลใบขนสินค้า (FCL เท่านั้น) เมื่อกรอก VAT ค่าของแล้ว
	// VAT จะใช้ตัวเลขจริงตามใบขนแทนการประมาณ 7%
	Customs *ImportCustoms `bson:"customs,omitempty" json:"customs,omitempty"`
	// VATSource is "customs" when VAT comes from the customs declaration, otherwise "estimated" (7%).
	VATSource string `bson:"vatSource" json:"vatSource"`
	// Items
	Items []ImportItem `bson:"items" json:"items"`
	// Summary
	TotalCBM          float64 `bson:"totalCBM" json:"totalCBM"`
	TotalShippingCost float64 `bson:"totalShippingCost" json:"totalShippingCost"`
	TotalProductCost  float64 `bson:"totalProductCost" json:"totalProductCost"`
	TotalImportDuty   float64 `bson:"totalImportDuty" json:"totalImportDuty"`
	TotalGoodsVAT     float64 `bson:"totalGoodsVAT" json:"totalGoodsVAT"`
	TotalFreightVAT   float64 `bson:"totalFreightVAT" json:"totalFreightVAT"`
	TotalVAT          float64 `bson:"totalVAT" json:"totalVAT"`
	GrandTotal        float64 `bson:"grandTotal" json:"grandTotal"`
	// Status
	Status        string  `bson:"status" json:"status"` // "draft" | "confirmed" | "purchased"
	PurchaseID    *string `bson:"purchaseId,omitempty" json:"purchaseId,omitempty"`
	PurchaseIsVAT *bool   `bson:"purchaseIsVAT,omitempty" json:"purchaseIsVAT,omitempty"`
	// PurchaseVATMode is the mode the purchase was created with: "vat" | "nonvat" | "import".
	PurchaseVATMode *string   `bson:"purchaseVatMode,omitempty" json:"purchaseVatMode,omitempty"`
	Notes           *string   `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time `bson:"updatedAt" json:"updatedAt"`
}

// ImportCustoms holds the figures from the customs declaration (ใบขนสินค้าขาเข้า) of this
// import. One declaration is issued per shipper, so it belongs to the import, not the container.
type ImportCustoms struct {
	DeclarationNo   string     `bson:"declarationNo" json:"declarationNo"`
	DeclarationDate *time.Time `bson:"declarationDate,omitempty" json:"declarationDate,omitempty"`
	CIFValue        float64    `bson:"cifValue" json:"cifValue"`         // ราคา CIF (บาท) ตามใบขน
	FreightValue    float64    `bson:"freightValue" json:"freightValue"` // ค่าเฟรท F (บาท) ตามใบขน
	ImportDuty      float64    `bson:"importDuty" json:"importDuty"`     // อากรขาเข้า (บาท)
	VAT             float64    `bson:"vat" json:"vat"`                   // ภาษีมูลค่าเพิ่ม (ยอดรวมตามใบขน)
}

// SplitVAT splits the declaration's VAT into VAT ค่าของ and VAT ค่าเฟรท in proportion to
// the VAT base: freight / (CIF + อากร). Insurance (I) stays with the goods.
func (c *ImportCustoms) SplitVAT() (goodsVAT, freightVAT float64) {
	if c == nil {
		return 0, 0
	}
	base := c.CIFValue + c.ImportDuty
	if base > 0 && c.FreightValue > 0 {
		freightVAT = utils.RoundTo2(c.VAT * c.FreightValue / base)
	}
	return utils.RoundTo2(c.VAT - freightVAT), freightVAT
}

// Import VAT modes for creating a purchase from an import.
const (
	ImportVATModeVAT    = "vat"
	ImportVATModeNonVAT = "nonvat"
	ImportVATModeImport = "import"
)

const (
	VATSourceEstimated = "estimated"
	VATSourceCustoms   = "customs"
)

type FCLCostDetail struct {
	Name   string  `bson:"name" json:"name"`
	Amount float64 `bson:"amount" json:"amount"`
}

type ImportItem struct {
	ProductID           string  `bson:"productId" json:"productId"`
	ProductName         string  `bson:"productName" json:"productName"`
	ProductCode         string  `bson:"productCode" json:"productCode"`
	UsdPricePerUnit     float64 `bson:"usdPricePerUnit" json:"usdPricePerUnit"`
	Quantity            int     `bson:"quantity" json:"quantity"`
	PiecesPerBox        int     `bson:"piecesPerBox" json:"piecesPerBox"`
	BoxWidth            float64 `bson:"boxWidth" json:"boxWidth"`
	BoxLength           float64 `bson:"boxLength" json:"boxLength"`
	BoxHeight           float64 `bson:"boxHeight" json:"boxHeight"`
	CBM                 float64 `bson:"cbm" json:"cbm"`
	CBMManual           bool    `bson:"cbmManual" json:"cbmManual"`
	ShippingCostPerUnit float64 `bson:"shippingCostPerUnit" json:"shippingCostPerUnit"`
	DutyPerUnit         float64 `bson:"dutyPerUnit" json:"dutyPerUnit"`
	Commission          float64 `bson:"commission" json:"commission"`
	CommissionPaid      bool    `bson:"commissionPaid" json:"commissionPaid"`
	// CostPerUnitBeforeVAT = ค่าของ + ค่าส่ง + อากร + ค่าคอม
	CostPerUnitBeforeVAT float64 `bson:"costPerUnitBeforeVAT" json:"costPerUnitBeforeVAT"`
	// GoodsVATPerUnit / FreightVATPerUnit are the customs VAT split (only when VATSource = "customs").
	GoodsVATPerUnit     float64 `bson:"goodsVATPerUnit" json:"goodsVATPerUnit"`
	FreightVATPerUnit   float64 `bson:"freightVATPerUnit" json:"freightVATPerUnit"`
	VATPerUnit          float64 `bson:"vatPerUnit" json:"vatPerUnit"`
	CostPerUnitAfterVAT float64 `bson:"costPerUnitAfterVAT" json:"costPerUnitAfterVAT"`
	TotalCost           float64 `bson:"totalCost" json:"totalCost"`
}

type InternationalImportRequest struct {
	ImportDate        time.Time       `json:"importDate"`
	ImportType        string          `json:"importType"`
	SupplierID        string          `json:"supplierId"`
	ShippingCompanyID string          `json:"shippingCompanyId"`
	UsdToThbRate      float64         `json:"usdToThbRate"`
	PricePerCBM       float64         `json:"pricePerCBM"`
	FCLCostDetails    []FCLCostDetail `json:"fclCostDetails"`
	FCLShipmentID     *string         `json:"fclShipmentId,omitempty"`
	Customs           *ImportCustoms  `json:"customs,omitempty"`
	Items             []ImportItem    `json:"items"`
	Notes             *string         `json:"notes,omitempty"`
}

// importCostParams describes how shipping and customs VAT are allocated to items.
type importCostParams struct {
	importType   string
	usdToThbRate float64
	pricePerCBM  float64 // LCL: shipping = item CBM × price per CBM
	// FCL: shipping = item CBM / cbmBase × shippingPool (cbmBase is the import's or the container's CBM)
	shippingPool float64
	cbmBase      float64
	customs      *ImportCustoms
}

// importTotals is the summary of an import after its items are costed.
type importTotals struct {
	cbm, shipping, product, duty, goodsVAT, freightVAT, vat, grand float64
	vatSource                                                      string
}

// itemCBM returns the CBM of an item, computing it from box dimensions unless entered manually.
func itemCBM(item *ImportItem) float64 {
	if item.CBMManual {
		return item.CBM
	}
	ppb := item.PiecesPerBox
	if ppb <= 0 {
		ppb = 1
	}
	numBoxes := math.Ceil(float64(item.Quantity) / float64(ppb))
	rawCBM := numBoxes * item.BoxWidth * item.BoxLength * item.BoxHeight / 1_000_000
	return math.Ceil(rawCBM*10) / 10
}

// customsApplies reports whether VAT should come from the customs declaration:
// FCL imports whose declaration VAT has been entered.
func customsApplies(importType string, customs *ImportCustoms) bool {
	return importType == "FCL" && customs != nil && customs.VAT > 0
}

// calculateImportItems fills shipping, duty, VAT and cost fields for every item.
//
// Customs mode (FCL with ใบขน): VAT ตามใบขนแยกเป็น VAT ค่าของ / VAT ค่าเฟรท (SplitVAT) แล้ว
//   - อากร และ VAT ค่าของ เฉลี่ยตามสัดส่วนมูลค่าสินค้า (ค่าของ THB) ภายใน imp
//   - VAT ค่าเฟรท เฉลี่ยตามสัดส่วน CBM ภายใน imp
//
// Otherwise VAT is estimated as 7% of (ค่าของ + ค่าส่ง + อากร); ค่าคอมไม่คิด VAT.
func calculateImportItems(items []ImportItem, p importCostParams) importTotals {
	useCustoms := customsApplies(p.importType, p.customs)

	var duty float64
	if p.importType == "FCL" && p.customs != nil {
		duty = p.customs.ImportDuty
	}

	var goodsVAT, freightVAT float64
	if useCustoms {
		goodsVAT, freightVAT = p.customs.SplitVAT()
	}

	totalGoods, importCBM := 0.0, 0.0
	for i := range items {
		items[i].CBM = itemCBM(&items[i])
		importCBM += items[i].CBM
		totalGoods += items[i].UsdPricePerUnit * p.usdToThbRate * float64(items[i].Quantity)
	}

	t := importTotals{vatSource: VATSourceEstimated}
	if useCustoms {
		t.vatSource = VATSourceCustoms
	}

	for i := range items {
		item := &items[i]
		qty := float64(item.Quantity)
		productCost := item.UsdPricePerUnit * p.usdToThbRate

		var shippingPerUnit, dutyPerUnit, goodsVATPerUnit, freightVATPerUnit float64
		if item.Quantity > 0 {
			if p.importType == "LCL" {
				shippingPerUnit = (item.CBM * p.pricePerCBM) / qty
			} else if p.importType == "FCL" && p.cbmBase > 0 {
				shippingPerUnit = (item.CBM / p.cbmBase) * p.shippingPool / qty
			}
			if totalGoods > 0 {
				goodsShare := productCost * qty / totalGoods
				dutyPerUnit = goodsShare * duty / qty
				goodsVATPerUnit = goodsShare * goodsVAT / qty
			}
			if importCBM > 0 {
				freightVATPerUnit = (item.CBM / importCBM) * freightVAT / qty
			}
		}

		item.ShippingCostPerUnit = shippingPerUnit
		item.DutyPerUnit = dutyPerUnit
		item.CostPerUnitBeforeVAT = productCost + shippingPerUnit + dutyPerUnit + item.Commission
		if useCustoms {
			item.GoodsVATPerUnit = goodsVATPerUnit
			item.FreightVATPerUnit = freightVATPerUnit
			item.VATPerUnit = goodsVATPerUnit + freightVATPerUnit
		} else {
			item.GoodsVATPerUnit = 0
			item.FreightVATPerUnit = 0
			// ค่าคอมไม่คิด VAT: VAT คิดจากค่าสินค้า + ค่าส่ง (+ อากร) เท่านั้น
			item.VATPerUnit = (productCost + shippingPerUnit + dutyPerUnit) * 0.07
		}
		item.CostPerUnitAfterVAT = item.CostPerUnitBeforeVAT + item.VATPerUnit
		item.TotalCost = item.CostPerUnitAfterVAT * qty

		t.cbm += item.CBM
		t.shipping += shippingPerUnit * qty
		t.product += productCost * qty
		t.duty += dutyPerUnit * qty
		t.goodsVAT += item.GoodsVATPerUnit * qty
		t.freightVAT += item.FreightVATPerUnit * qty
		t.vat += item.VATPerUnit * qty
		t.grand += item.TotalCost
	}
	return t
}

func (imp *InternationalImport) applyTotals(t importTotals) {
	imp.TotalCBM = t.cbm
	imp.TotalShippingCost = t.shipping
	imp.TotalProductCost = t.product
	imp.TotalImportDuty = t.duty
	imp.TotalGoodsVAT = t.goodsVAT
	imp.TotalFreightVAT = t.freightVAT
	imp.TotalVAT = t.vat
	imp.GrandTotal = t.grand
	imp.VATSource = t.vatSource
}

// standaloneCostParams are the allocation params for an import that is not linked to a
// container: FCL shipping and freight VAT are spread over the import's own CBM.
func (r *InternationalImportRequest) standaloneCostParams() importCostParams {
	totalFCLCost := 0.0
	for _, d := range r.FCLCostDetails {
		totalFCLCost += d.Amount
	}
	totalCBM := 0.0
	for i := range r.Items {
		totalCBM += itemCBM(&r.Items[i])
	}
	return importCostParams{
		importType:   r.ImportType,
		usdToThbRate: r.UsdToThbRate,
		pricePerCBM:  r.PricePerCBM,
		shippingPool: totalFCLCost,
		cbmBase:      totalCBM,
		customs:      r.Customs,
	}
}

// CalculateItemCosts recalculates CBM, shipping, and cost fields for all items.
func (r *InternationalImportRequest) CalculateItemCosts() importTotals {
	if r.ImportType != "FCL" {
		r.Customs = nil
	}
	return calculateImportItems(r.Items, r.standaloneCostParams())
}

// RecalcItemCBM recomputes CBM for every item and returns the total CBM of this import.
// This is the import's contribution to its FCL container's total CBM.
func (imp *InternationalImport) RecalcItemCBM() float64 {
	totalCBM := 0.0
	for i := range imp.Items {
		item := &imp.Items[i]
		item.CBM = itemCBM(item)
		totalCBM += item.CBM
	}
	return totalCBM
}

// RecalculateForFCLContainer recomputes all item and summary costs for an import that
// belongs to an FCL container. Shipping per unit is allocated by CBM share of the whole
// container: (item CBM / container total CBM) × container total cost / quantity.
// containerCBM is the CBM summed across every import linked to the same container.
func (imp *InternationalImport) RecalculateForFCLContainer(containerCost, containerCBM float64) {
	t := calculateImportItems(imp.Items, importCostParams{
		importType:   "FCL",
		usdToThbRate: imp.UsdToThbRate,
		shippingPool: containerCost,
		cbmBase:      containerCBM,
		customs:      imp.Customs,
	})
	imp.applyTotals(t)
	imp.UpdatedAt = utils.NowInThailand()
}

func (r *InternationalImportRequest) ToInternationalImport() *InternationalImport {
	now := utils.NowInThailand()

	totals := r.CalculateItemCosts()

	totalFCLCost := 0.0
	for _, d := range r.FCLCostDetails {
		totalFCLCost += d.Amount
	}

	fclDetails := r.FCLCostDetails
	if fclDetails == nil {
		fclDetails = []FCLCostDetail{}
	}
	items := r.Items
	if items == nil {
		items = []ImportItem{}
	}

	imp := &InternationalImport{
		ImportCode:          "",
		ImportDate:          r.ImportDate,
		ImportType:          r.ImportType,
		SupplierID:          r.SupplierID,
		SupplierName:        "",
		ShippingCompanyID:   r.ShippingCompanyID,
		ShippingCompanyName: "",
		UsdToThbRate:        r.UsdToThbRate,
		PricePerCBM:         r.PricePerCBM,
		FCLCostDetails:      fclDetails,
		TotalFCLCost:        totalFCLCost,
		FCLShipmentID:       r.FCLShipmentID,
		Customs:             r.Customs,
		Items:               items,
		Status:              "draft",
		Notes:               r.Notes,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	imp.applyTotals(totals)
	return imp
}

func (imp *InternationalImport) UpdateFromRequest(r *InternationalImportRequest) {
	totals := r.CalculateItemCosts()

	totalFCLCost := 0.0
	for _, d := range r.FCLCostDetails {
		totalFCLCost += d.Amount
	}

	fclDetails := r.FCLCostDetails
	if fclDetails == nil {
		fclDetails = []FCLCostDetail{}
	}
	items := r.Items
	if items == nil {
		items = []ImportItem{}
	}

	imp.ImportDate = r.ImportDate
	imp.ImportType = r.ImportType
	imp.SupplierID = r.SupplierID
	imp.ShippingCompanyID = r.ShippingCompanyID
	imp.UsdToThbRate = r.UsdToThbRate
	imp.PricePerCBM = r.PricePerCBM
	imp.FCLCostDetails = fclDetails
	imp.TotalFCLCost = totalFCLCost
	imp.FCLShipmentID = r.FCLShipmentID
	imp.Customs = r.Customs
	imp.Items = items
	imp.applyTotals(totals)
	imp.Notes = r.Notes
	imp.UpdatedAt = utils.NowInThailand()
}

// CanCreateImportPurchase reports whether a purchase "นำเข้า (VAT ตามใบขน)" can be created:
// only FCL imports whose customs VAT has been entered.
func (imp *InternationalImport) CanCreateImportPurchase() bool {
	return imp.ImportType == "FCL" && imp.VATSource == VATSourceCustoms
}

// ToPurchaseRequest converts the import to a PurchaseRequest for creating a domestic purchase record.
// mode is ImportVATModeVAT (VAT นอก 7%), ImportVATModeNonVAT or ImportVATModeImport (VAT ตามใบขน).
func (imp *InternationalImport) ToPurchaseRequest(mode string) *PurchaseRequest {
	isImport := mode == ImportVATModeImport
	isVAT := mode != ImportVATModeNonVAT

	purchaseItems := make([]PurchaseItem, len(imp.Items))
	var lineVATSum float64
	lastVATLine := -1
	for i, item := range imp.Items {
		// ใช้ราคาก่อน VAT เสมอ: ถ้ามี VAT จะสร้างเป็น VAT นอก (ราคา + VAT 7%)
		// ค่าคอมไม่คิด VAT จึงแยกออกจากราคาต่อชิ้น แต่ยังนับเป็นต้นทุนจริงของสินค้า
		unitPrice := item.CostPerUnitBeforeVAT - item.Commission
		var commissionPerUnit *float64
		if item.Commission != 0 {
			c := item.Commission
			commissionPerUnit = &c
		}
		purchaseItems[i] = PurchaseItem{
			ProductID:         item.ProductID,
			ProductName:       item.ProductName,
			ProductCode:       item.ProductCode,
			Quantity:          item.Quantity,
			UnitPrice:         unitPrice,
			CommissionPerUnit: commissionPerUnit,
			TotalPrice:        unitPrice * float64(item.Quantity),
		}
		if isImport {
			// VAT จริงตามใบขนของทั้งรายการ
			lineVAT := utils.RoundTo2(item.VATPerUnit * float64(item.Quantity))
			purchaseItems[i].ImportVAT = &lineVAT
			lineVATSum += lineVAT
			if item.Quantity > 0 {
				lastVATLine = i
			}
		}
	}

	vatType := ""
	var customs *ImportCustomsInfo
	if isImport {
		vatType = VATTypeImport
		// ปัดเศษให้ VAT รวมของใบซื้อตรงกับใบขนทุกสตางค์: ใส่ส่วนต่างที่รายการสุดท้าย
		want := utils.RoundTo2(imp.TotalVAT)
		if lastVATLine >= 0 {
			if diff := utils.RoundTo2(want - lineVATSum); diff != 0 {
				adjusted := utils.RoundTo2(*purchaseItems[lastVATLine].ImportVAT + diff)
				purchaseItems[lastVATLine].ImportVAT = &adjusted
			}
		}
		customs = &ImportCustomsInfo{
			ImportID:   imp.ID.Hex(),
			ImportCode: imp.ImportCode,
			GoodsVAT:   utils.RoundTo2(imp.TotalGoodsVAT),
			FreightVAT: utils.RoundTo2(imp.TotalFreightVAT),
			ImportDuty: utils.RoundTo2(imp.TotalImportDuty),
		}
		if imp.Customs != nil {
			customs.DeclarationNo = imp.Customs.DeclarationNo
			customs.DeclarationDate = imp.Customs.DeclarationDate
		}
	} else if isVAT {
		vatType = "exclusive"
	}

	return &PurchaseRequest{
		PurchaseDate:  imp.ImportDate,
		SupplierID:    imp.SupplierID,
		Items:         purchaseItems,
		IsVAT:         isVAT,
		VATType:       vatType,
		ImportCustoms: customs,
		ShippingCost:  0,
		Payment: PaymentInfo{
			IsPaid: false,
		},
		Warehouse: WarehouseInfo{
			IsUpdated: false,
			Items:     []WarehouseItem{},
		},
	}
}
