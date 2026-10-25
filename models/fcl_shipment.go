package models

import (
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FCLShipment represents one full container (เหมาตู้). A single container can hold
// goods from several factories, so multiple InternationalImport records (each with
// its own imp id / supplier) link to one FCLShipment via FCLShipmentID.
// The container is the single source of truth for shipping cost; each linked import's
// per-unit shipping is derived from the container total, allocated by CBM share.
type FCLShipment struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FCLCode             string             `bson:"fclCode" json:"fclCode"`
	ShipmentDate        time.Time          `bson:"shipmentDate" json:"shipmentDate"`
	ShippingCompanyID   string             `bson:"shippingCompanyId" json:"shippingCompanyId"`
	ShippingCompanyName string             `bson:"shippingCompanyName" json:"shippingCompanyName"`
	UsdToThbRate        float64            `bson:"usdToThbRate" json:"usdToThbRate"` // default rate for USD cost lines
	CostLines           []FCLCostLine      `bson:"costLines" json:"costLines"`
	TotalCostThb        float64            `bson:"totalCostThb" json:"totalCostThb"`
	// TotalVAT is the input VAT (ภาษีซื้อ) on the cost lines' tax invoices. It is not part of
	// TotalCostThb and is not allocated to products.
	TotalVAT float64 `bson:"totalVAT" json:"totalVAT"`
	// TotalCBM and LinkedImportCount are summed across every linked import and refreshed
	// whenever the container is recomputed.
	TotalCBM          float64   `bson:"totalCBM" json:"totalCBM"`
	LinkedImportCount int       `bson:"linkedImportCount" json:"linkedImportCount"`
	Status            string    `bson:"status" json:"status"` // "open" | "closed"
	Notes             *string   `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt         time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time `bson:"updatedAt" json:"updatedAt"`
}

// FCLCostLine is one charge that makes up the container cost (freight, THC, clearing...).
// Each line may be entered in THB or USD; USD lines are converted to THB using the line's
// own UsdRate, falling back to the shipment's UsdToThbRate when not provided.
// Amount is before VAT. VAT (THB) comes from the vendor's tax invoice (ใบกำกับภาษี) and
// counts as input VAT, not as container cost.
type FCLCostLine struct {
	Name           string     `bson:"name" json:"name"`
	Currency       string     `bson:"currency" json:"currency"` // "THB" | "USD"
	Amount         float64    `bson:"amount" json:"amount"`
	UsdRate        float64    `bson:"usdRate" json:"usdRate"`
	AmountThb      float64    `bson:"amountThb" json:"amountThb"`
	VAT            float64    `bson:"vat" json:"vat"`
	Vendor         string     `bson:"vendor" json:"vendor"`
	TaxInvoiceNo   string     `bson:"taxInvoiceNo" json:"taxInvoiceNo"`
	TaxInvoiceDate *time.Time `bson:"taxInvoiceDate,omitempty" json:"taxInvoiceDate,omitempty"`
}

type FCLShipmentRequest struct {
	ShipmentDate      time.Time     `json:"shipmentDate"`
	ShippingCompanyID string        `json:"shippingCompanyId"`
	UsdToThbRate      float64       `json:"usdToThbRate"`
	CostLines         []FCLCostLine `json:"costLines"`
	Notes             *string       `json:"notes,omitempty"`
}

// normalizeCostLines fills AmountThb for each line and returns the lines plus the totals
// in THB: cost before VAT and VAT.
func normalizeCostLines(lines []FCLCostLine, defaultRate float64) ([]FCLCostLine, float64, float64) {
	out := make([]FCLCostLine, 0, len(lines))
	total, totalVAT := 0.0, 0.0
	for _, l := range lines {
		line := l
		if line.Currency == "USD" {
			rate := line.UsdRate
			if rate <= 0 {
				rate = defaultRate
			}
			line.UsdRate = rate
			line.AmountThb = line.Amount * rate
		} else {
			line.Currency = "THB"
			line.UsdRate = 0
			line.AmountThb = line.Amount
		}
		total += line.AmountThb
		totalVAT += line.VAT
		out = append(out, line)
	}
	return out, total, utils.RoundTo2(totalVAT)
}

func (r *FCLShipmentRequest) ToFCLShipment() *FCLShipment {
	now := utils.NowInThailand()
	lines, total, totalVAT := normalizeCostLines(r.CostLines, r.UsdToThbRate)
	return &FCLShipment{
		ShipmentDate:      r.ShipmentDate,
		ShippingCompanyID: r.ShippingCompanyID,
		UsdToThbRate:      r.UsdToThbRate,
		CostLines:         lines,
		TotalCostThb:      total,
		TotalVAT:          totalVAT,
		TotalCBM:          0,
		LinkedImportCount: 0,
		Status:            "open",
		Notes:             r.Notes,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// UpdateFromRequest applies editable fields from the request. Status, totals derived
// from linked imports, and identity fields are managed elsewhere.
func (s *FCLShipment) UpdateFromRequest(r *FCLShipmentRequest) {
	lines, total, totalVAT := normalizeCostLines(r.CostLines, r.UsdToThbRate)
	s.ShipmentDate = r.ShipmentDate
	s.ShippingCompanyID = r.ShippingCompanyID
	s.UsdToThbRate = r.UsdToThbRate
	s.CostLines = lines
	s.TotalCostThb = total
	s.TotalVAT = totalVAT
	s.Notes = r.Notes
	s.UpdatedAt = utils.NowInThailand()
}
