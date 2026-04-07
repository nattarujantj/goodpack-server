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
	// Items
	Items []ImportItem `bson:"items" json:"items"`
	// Summary
	TotalCBM          float64 `bson:"totalCBM" json:"totalCBM"`
	TotalShippingCost float64 `bson:"totalShippingCost" json:"totalShippingCost"`
	TotalProductCost  float64 `bson:"totalProductCost" json:"totalProductCost"`
	GrandTotal        float64 `bson:"grandTotal" json:"grandTotal"`
	// Status
	Status     string  `bson:"status" json:"status"` // "draft" | "purchased"
	PurchaseID *string `bson:"purchaseId,omitempty" json:"purchaseId,omitempty"`
	Notes      *string `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt" json:"updatedAt"`
}

type FCLCostDetail struct {
	Name   string  `bson:"name" json:"name"`
	Amount float64 `bson:"amount" json:"amount"`
}

type ImportItem struct {
	ProductID            string  `bson:"productId" json:"productId"`
	ProductName          string  `bson:"productName" json:"productName"`
	ProductCode          string  `bson:"productCode" json:"productCode"`
	UsdPricePerUnit      float64 `bson:"usdPricePerUnit" json:"usdPricePerUnit"`
	Quantity             int     `bson:"quantity" json:"quantity"`
	PiecesPerBox         int     `bson:"piecesPerBox" json:"piecesPerBox"`
	BoxWidth             float64 `bson:"boxWidth" json:"boxWidth"`
	BoxLength            float64 `bson:"boxLength" json:"boxLength"`
	BoxHeight            float64 `bson:"boxHeight" json:"boxHeight"`
	CBM                  float64 `bson:"cbm" json:"cbm"`
	ShippingCostPerUnit  float64 `bson:"shippingCostPerUnit" json:"shippingCostPerUnit"`
	Commission           float64 `bson:"commission" json:"commission"`
	CostPerUnitBeforeVAT float64 `bson:"costPerUnitBeforeVAT" json:"costPerUnitBeforeVAT"`
	VATPerUnit           float64 `bson:"vatPerUnit" json:"vatPerUnit"`
	CostPerUnitAfterVAT  float64 `bson:"costPerUnitAfterVAT" json:"costPerUnitAfterVAT"`
	TotalCost            float64 `bson:"totalCost" json:"totalCost"`
}

type InternationalImportRequest struct {
	ImportDate        time.Time       `json:"importDate"`
	ImportType        string          `json:"importType"`
	SupplierID        string          `json:"supplierId"`
	ShippingCompanyID string          `json:"shippingCompanyId"`
	UsdToThbRate      float64         `json:"usdToThbRate"`
	PricePerCBM       float64         `json:"pricePerCBM"`
	FCLCostDetails    []FCLCostDetail `json:"fclCostDetails"`
	Items             []ImportItem    `json:"items"`
	Notes             *string         `json:"notes,omitempty"`
}

// CalculateItemCosts recalculates CBM, shipping, and cost fields for all items.
func (r *InternationalImportRequest) CalculateItemCosts() {
	totalFCLCost := 0.0
	for _, d := range r.FCLCostDetails {
		totalFCLCost += d.Amount
	}

	// First pass: calculate CBM for all items
	totalCBM := 0.0
	for i := range r.Items {
		item := &r.Items[i]
		ppb := item.PiecesPerBox
		if ppb <= 0 {
			ppb = 1
		}
		numBoxes := math.Ceil(float64(item.Quantity) / float64(ppb))
		rawCBM := numBoxes * item.BoxWidth * item.BoxLength * item.BoxHeight / 1_000_000
		item.CBM = math.Ceil(rawCBM)
		totalCBM += item.CBM
	}

	// Second pass: calculate costs
	for i := range r.Items {
		item := &r.Items[i]
		productCost := item.UsdPricePerUnit * r.UsdToThbRate

		var shippingPerUnit float64
		if r.ImportType == "LCL" {
			if item.Quantity > 0 {
				shippingPerUnit = (item.CBM * r.PricePerCBM) / float64(item.Quantity)
			}
		} else if r.ImportType == "FCL" {
			if totalCBM > 0 && item.Quantity > 0 {
				shippingPerUnit = (item.CBM / totalCBM) * totalFCLCost / float64(item.Quantity)
			}
		}

		item.ShippingCostPerUnit = shippingPerUnit
		item.CostPerUnitBeforeVAT = productCost + shippingPerUnit + item.Commission
		item.VATPerUnit = item.CostPerUnitBeforeVAT * 0.07
		item.CostPerUnitAfterVAT = item.CostPerUnitBeforeVAT + item.VATPerUnit
		item.TotalCost = item.CostPerUnitAfterVAT * float64(item.Quantity)
	}
}

func (r *InternationalImportRequest) ToInternationalImport() *InternationalImport {
	now := utils.NowInThailand()

	r.CalculateItemCosts()

	totalFCLCost := 0.0
	for _, d := range r.FCLCostDetails {
		totalFCLCost += d.Amount
	}

	var totalCBM, totalShippingCost, totalProductCost, grandTotal float64
	for _, item := range r.Items {
		totalCBM += item.CBM
		totalShippingCost += item.ShippingCostPerUnit * float64(item.Quantity)
		totalProductCost += item.UsdPricePerUnit * r.UsdToThbRate * float64(item.Quantity)
		grandTotal += item.TotalCost
	}

	fclDetails := r.FCLCostDetails
	if fclDetails == nil {
		fclDetails = []FCLCostDetail{}
	}
	items := r.Items
	if items == nil {
		items = []ImportItem{}
	}

	return &InternationalImport{
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
		Items:               items,
		TotalCBM:            totalCBM,
		TotalShippingCost:   totalShippingCost,
		TotalProductCost:    totalProductCost,
		GrandTotal:          grandTotal,
		Status:              "draft",
		Notes:               r.Notes,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func (imp *InternationalImport) UpdateFromRequest(r *InternationalImportRequest) {
	r.CalculateItemCosts()

	totalFCLCost := 0.0
	for _, d := range r.FCLCostDetails {
		totalFCLCost += d.Amount
	}

	var totalCBM, totalShippingCost, totalProductCost, grandTotal float64
	for _, item := range r.Items {
		totalCBM += item.CBM
		totalShippingCost += item.ShippingCostPerUnit * float64(item.Quantity)
		totalProductCost += item.UsdPricePerUnit * r.UsdToThbRate * float64(item.Quantity)
		grandTotal += item.TotalCost
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
	imp.Items = items
	imp.TotalCBM = totalCBM
	imp.TotalShippingCost = totalShippingCost
	imp.TotalProductCost = totalProductCost
	imp.GrandTotal = grandTotal
	imp.Notes = r.Notes
	imp.UpdatedAt = utils.NowInThailand()
}

// ToPurchaseRequest converts the import to a PurchaseRequest for creating a domestic purchase record.
func (imp *InternationalImport) ToPurchaseRequest() *PurchaseRequest {
	purchaseItems := make([]PurchaseItem, len(imp.Items))
	for i, item := range imp.Items {
		purchaseItems[i] = PurchaseItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity,
			UnitPrice:   item.CostPerUnitBeforeVAT,
			TotalPrice:  item.CostPerUnitBeforeVAT * float64(item.Quantity),
		}
	}

	return &PurchaseRequest{
		PurchaseDate: imp.ImportDate,
		SupplierID:   imp.SupplierID,
		Items:        purchaseItems,
		IsVAT:        true,
		VATType:      "exclusive",
		ShippingCost: 0,
		Payment: PaymentInfo{
			IsPaid: false,
		},
		Warehouse: WarehouseInfo{
			IsUpdated: false,
			Items:     []WarehouseItem{},
		},
	}
}
