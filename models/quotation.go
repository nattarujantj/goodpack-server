package models

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CustomTime handles ISO 8601 datetime format
type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1] // Remove quotes

	// Try different time formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			ct.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse time: %s", s)
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.Time.Format(time.RFC3339))
}

// QuotationItem represents an item in a quotation
type QuotationItem struct {
	ProductID   string  `bson:"productId" json:"productId"`     // รหัสสินค้า
	ProductName string  `bson:"productName" json:"productName"` // ชื่อสินค้า
	ProductCode string  `bson:"productCode" json:"productCode"` // รหัสสินค้า
	Quantity    int     `bson:"quantity" json:"quantity"`       // จำนวน
	UnitPrice   float64 `bson:"unitPrice" json:"unitPrice"`     // ราคาต่อหน่วย
	TotalPrice  float64 `bson:"totalPrice" json:"totalPrice"`   // ราคารวม
}

// Quotation represents a quotation document
type Quotation struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	QuotationCode     string             `bson:"quotationCode" json:"quotationCode"`                             // QU-YYMM-XXXX
	QuotationDate     time.Time          `bson:"quotationDate" json:"quotationDate"`                             // วันที่เสนอราคา
	CustomerID        string             `bson:"customerId" json:"customerId"`                                   // รหัสลูกค้า
	CustomerName      string             `bson:"customerName" json:"customerName"`                               // ชื่อลูกค้า
	ContactName       *string            `bson:"contactName,omitempty" json:"contactName,omitempty"`             // ชื่อผู้ติดต่อ
	CustomerCode      *string            `bson:"customerCode,omitempty" json:"customerCode,omitempty"`           // รหัสลูกค้า
	TaxID             *string            `bson:"taxId,omitempty" json:"taxId,omitempty"`                         // เลขประจำตัวผู้เสียภาษี
	Address           *string            `bson:"address,omitempty" json:"address,omitempty"`                     // ที่อยู่
	Phone             *string            `bson:"phone,omitempty" json:"phone,omitempty"`                         // เบอร์โทรศัพท์
	Items             []QuotationItem    `bson:"items" json:"items"`                                             // รายการสินค้า
	IsVAT             bool               `bson:"isVAT" json:"isVAT"`                                             // มี VAT หรือไม่
	ShippingCost      float64            `bson:"shippingCost" json:"shippingCost"`                               // ค่าขนส่ง
	Notes             *string            `bson:"notes,omitempty" json:"notes,omitempty"`                         // หมายเหตุ
	ValidUntil        *time.Time         `bson:"validUntil,omitempty" json:"validUntil,omitempty"`               // ราคาใช้ได้ถึง
	Status            string             `bson:"status" json:"status"`                                           // สถานะ (draft, sent, accepted, rejected, expired)
	SaleCode          *string            `bson:"saleCode,omitempty" json:"saleCode,omitempty"`                   // รหัสรายการขายที่สร้างจาก quotation นี้
	BankAccountID     *string            `bson:"bankAccountId,omitempty" json:"bankAccountId,omitempty"`         // รหัสบัญชีธนาคาร
	BankName          *string            `bson:"bankName,omitempty" json:"bankName,omitempty"`                   // ชื่อธนาคาร
	BankAccountName   *string            `bson:"bankAccountName,omitempty" json:"bankAccountName,omitempty"`     // ชื่อบัญชี
	BankAccountNumber *string            `bson:"bankAccountNumber,omitempty" json:"bankAccountNumber,omitempty"` // เลขบัญชี
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// QuotationRequest represents the request body for creating/updating a quotation
type QuotationRequest struct {
	QuotationDate     CustomTime      `json:"quotationDate"`
	CustomerID        string          `json:"customerId"`
	Items             []QuotationItem `json:"items"`
	IsVAT             bool            `json:"isVAT"`
	ShippingCost      float64         `json:"shippingCost"`
	Notes             *string         `json:"notes,omitempty"`
	ValidUntil        *CustomTime     `json:"validUntil,omitempty"`
	Status            string          `json:"status"`
	BankAccountID     *string         `json:"bankAccountId,omitempty"`
	BankName          *string         `json:"bankName,omitempty"`
	BankAccountName   *string         `json:"bankAccountName,omitempty"`
	BankAccountNumber *string         `json:"bankAccountNumber,omitempty"`
}

// ToQuotation converts QuotationRequest to Quotation
func (qr *QuotationRequest) ToQuotation() *Quotation {
	now := time.Now()
	quotation := &Quotation{
		QuotationDate:     qr.QuotationDate.Time,
		CustomerID:        qr.CustomerID,
		Items:             qr.Items,
		IsVAT:             qr.IsVAT,
		ShippingCost:      qr.ShippingCost,
		Notes:             qr.Notes,
		Status:            qr.Status,
		BankAccountID:     qr.BankAccountID,
		BankName:          qr.BankName,
		BankAccountName:   qr.BankAccountName,
		BankAccountNumber: qr.BankAccountNumber,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if qr.ValidUntil != nil {
		quotation.ValidUntil = &qr.ValidUntil.Time
	}

	return quotation
}

// UpdateFromRequest updates Quotation from QuotationRequest
func (q *Quotation) UpdateFromRequest(qr *QuotationRequest) {
	q.QuotationDate = qr.QuotationDate.Time
	q.CustomerID = qr.CustomerID
	q.Items = qr.Items
	q.IsVAT = qr.IsVAT
	q.ShippingCost = qr.ShippingCost
	q.Notes = qr.Notes
	if qr.ValidUntil != nil {
		q.ValidUntil = &qr.ValidUntil.Time
	} else {
		q.ValidUntil = nil
	}
	q.Status = qr.Status
	q.BankAccountID = qr.BankAccountID
	q.BankName = qr.BankName
	q.BankAccountName = qr.BankAccountName
	q.BankAccountNumber = qr.BankAccountNumber
	q.UpdatedAt = time.Now()
}

// GenerateQuotationCode generates a new quotation code in format QU-YYMM-XXXX
func GenerateQuotationCode(lastCode string) (string, error) {
	now := time.Now()
	buddhistYear := now.Year() + 543 // Convert to Buddhist year
	month := int(now.Month())

	prefix := fmt.Sprintf("QU-%02d%02d-", buddhistYear%100, month) // YYMM

	if lastCode == "" {
		return prefix + "0001", nil
	}

	// Extract the numeric part (XXXX)
	var lastYear, lastMonth, lastSeq int
	_, err := fmt.Sscanf(lastCode, "QU-%02d%02d-%04d", &lastYear, &lastMonth, &lastSeq)
	if err != nil {
		return "", fmt.Errorf("invalid last quotation code format: %w", err)
	}

	newSeq := lastSeq + 1
	return fmt.Sprintf("%s%04d", prefix, newSeq), nil
}

// CalculateGrandTotal calculates the grand total including VAT and shipping
func (q *Quotation) CalculateGrandTotal() float64 {
	totalBeforeVAT := 0.0
	for _, item := range q.Items {
		totalBeforeVAT += item.TotalPrice
	}

	totalVAT := 0.0
	if q.IsVAT {
		totalVAT = totalBeforeVAT * 0.07
	}

	return totalBeforeVAT + totalVAT + q.ShippingCost
}

// ToSaleRequest converts Quotation to SaleRequest for copying to sale
func (q *Quotation) ToSaleRequest() *SaleRequest {
	// Convert QuotationItem to SaleItem
	saleItems := make([]SaleItem, len(q.Items))
	for i, item := range q.Items {
		saleItems[i] = SaleItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			TotalPrice:  item.TotalPrice,
		}
	}

	return &SaleRequest{
		SaleDate:     time.Now(), // Use current date for sale
		CustomerID:   q.CustomerID,
		Items:        saleItems,
		IsVAT:        q.IsVAT,
		ShippingCost: q.ShippingCost,
		Payment: PaymentInfo{
			IsPaid: false, // Default to unpaid
		},
		Warehouse: WarehouseInfo{
			IsUpdated:      false,
			ActualShipping: q.ShippingCost,
			Items:          []WarehouseItem{}, // Empty warehouse items
		},
		Notes:             q.Notes,
		QuotationCode:     &q.QuotationCode, // Reference to original quotation
		BankAccountID:     q.BankAccountID,
		BankName:          q.BankName,
		BankAccountName:   q.BankAccountName,
		BankAccountNumber: q.BankAccountNumber,
	}
}
