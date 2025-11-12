package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StockAdjustmentType represents the type of stock adjustment
type StockAdjustmentType string

const (
	AdjustmentTypeAdd    StockAdjustmentType = "add"    // เพิ่ม
	AdjustmentTypeReduce StockAdjustmentType = "reduce" // ลด
)

// StockType represents which stock field is being adjusted
type StockType string

const (
	StockTypeVAT         StockType = "vat"         // VAT Stock
	StockTypeNonVAT      StockType = "nonvat"      // Non-VAT Stock
	StockTypeActualStock StockType = "actualstock" // Actual Stock
)

// SourceType represents where the stock change came from
type SourceType string

const (
	SourceTypePurchase   SourceType = "purchase"   // จากรายการซื้อ
	SourceTypeSale       SourceType = "sale"       // จากรายการขาย
	SourceTypeAdjustment SourceType = "adjustment" // จากฟีเจอร์แก้ไขสต็อก
	SourceTypeMigration  SourceType = "migration"  // จาก migration
)

// StockAdjustment represents a stock adjustment record
type StockAdjustment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID   string             `bson:"productId" json:"productId"`     // Product ID
	ProductName string             `bson:"productName" json:"productName"` // Product name (for display)
	SKUID       string             `bson:"skuId" json:"skuId"`             // SKU ID (for display)

	// Adjustment details
	AdjustmentType StockAdjustmentType `bson:"adjustmentType" json:"adjustmentType"` // add or reduce
	StockType      StockType           `bson:"stockType" json:"stockType"`           // vat, nonvat, or actualstock
	Quantity       int                 `bson:"quantity" json:"quantity"`             // จำนวนที่เพิ่ม/ลด

	// Stock values before and after adjustment
	BeforeVATPurchased    int `bson:"beforeVATPurchased" json:"beforeVATPurchased"`
	BeforeVATSold         int `bson:"beforeVATSold" json:"beforeVATSold"`
	BeforeVATRemaining    int `bson:"beforeVATRemaining" json:"beforeVATRemaining"`
	BeforeNonVATPurchased int `bson:"beforeNonVATPurchased" json:"beforeNonVATPurchased"`
	BeforeNonVATSold      int `bson:"beforeNonVATSold" json:"beforeNonVATSold"`
	BeforeNonVATRemaining int `bson:"beforeNonVATRemaining" json:"beforeNonVATRemaining"`
	BeforeActualStock     int `bson:"beforeActualStock" json:"beforeActualStock"`

	AfterVATPurchased    int `bson:"afterVATPurchased" json:"afterVATPurchased"`
	AfterVATSold         int `bson:"afterVATSold" json:"afterVATSold"`
	AfterVATRemaining    int `bson:"afterVATRemaining" json:"afterVATRemaining"`
	AfterNonVATPurchased int `bson:"afterNonVATPurchased" json:"afterNonVATPurchased"`
	AfterNonVATSold      int `bson:"afterNonVATSold" json:"afterNonVATSold"`
	AfterNonVATRemaining int `bson:"afterNonVATRemaining" json:"afterNonVATRemaining"`
	AfterActualStock     int `bson:"afterActualStock" json:"afterActualStock"`

	// Source information
	SourceType SourceType `bson:"sourceType" json:"sourceType"`                     // purchase, sale, adjustment, migration
	SourceID   *string    `bson:"sourceId,omitempty" json:"sourceId,omitempty"`     // ID of purchase/sale if applicable
	SourceCode *string    `bson:"sourceCode,omitempty" json:"sourceCode,omitempty"` // Code of purchase/sale (e.g., PUR-VAT-6701-0001)

	// Notes
	Notes *string `bson:"notes,omitempty" json:"notes,omitempty"` // หมายเหตุการแก้ไข

	// Metadata
	CreatedBy *string   `bson:"createdBy,omitempty" json:"createdBy,omitempty"` // User who made the adjustment
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// StockAdjustmentRequest represents the request body for creating a stock adjustment
type StockAdjustmentRequest struct {
	AdjustmentType StockAdjustmentType `json:"adjustmentType"`  // "add" or "reduce"
	StockType      StockType           `json:"stockType"`       // "vat", "nonvat", or "actualstock"
	Quantity       int                 `json:"quantity"`        // จำนวนที่เพิ่ม/ลด
	Notes          *string             `json:"notes,omitempty"` // หมายเหตุ
}

// ToStockAdjustment converts StockAdjustmentRequest to StockAdjustment
func (req *StockAdjustmentRequest) ToStockAdjustment(product *Product, sourceType SourceType, sourceID, sourceCode *string) *StockAdjustment {
	now := time.Now()

	adjustment := &StockAdjustment{
		ProductID:      product.ID.Hex(),
		ProductName:    product.Name,
		SKUID:          product.SKUID,
		AdjustmentType: req.AdjustmentType,
		StockType:      req.StockType,
		Quantity:       req.Quantity,
		SourceType:     sourceType,
		SourceID:       sourceID,
		SourceCode:     sourceCode,
		Notes:          req.Notes,
		CreatedAt:      now,
	}

	// Store before values
	adjustment.BeforeVATPurchased = product.Stock.VAT.Purchased
	adjustment.BeforeVATSold = product.Stock.VAT.Sold
	adjustment.BeforeVATRemaining = product.Stock.VAT.Remaining
	adjustment.BeforeNonVATPurchased = product.Stock.NonVAT.Purchased
	adjustment.BeforeNonVATSold = product.Stock.NonVAT.Sold
	adjustment.BeforeNonVATRemaining = product.Stock.NonVAT.Remaining
	adjustment.BeforeActualStock = product.Stock.ActualStock

	return adjustment
}

// SetAfterValues sets the after values from the updated product
func (sa *StockAdjustment) SetAfterValues(product *Product) {
	sa.AfterVATPurchased = product.Stock.VAT.Purchased
	sa.AfterVATSold = product.Stock.VAT.Sold
	sa.AfterVATRemaining = product.Stock.VAT.Remaining
	sa.AfterNonVATPurchased = product.Stock.NonVAT.Purchased
	sa.AfterNonVATSold = product.Stock.NonVAT.Sold
	sa.AfterNonVATRemaining = product.Stock.NonVAT.Remaining
	sa.AfterActualStock = product.Stock.ActualStock
}
