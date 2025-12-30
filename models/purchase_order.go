package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"goodpack-server/utils"
)

// PurchaseOrderItem represents an item in a purchase order
type PurchaseOrderItem struct {
	ProductID   string  `bson:"productId" json:"productId"`     // รหัสสินค้า
	ProductName string  `bson:"productName" json:"productName"` // ชื่อสินค้า
	ProductCode string  `bson:"productCode" json:"productCode"` // รหัสสินค้า
	Quantity    int     `bson:"quantity" json:"quantity"`       // จำนวน
	UnitPrice   float64 `bson:"unitPrice" json:"unitPrice"`     // ราคาต่อหน่วย
	TotalPrice  float64 `bson:"totalPrice" json:"totalPrice"`   // ราคารวม
}

// PurchaseOrder represents a purchase order document
type PurchaseOrder struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PurchaseOrderCode string             `bson:"purchaseOrderCode" json:"purchaseOrderCode"`                             // PO-YYMM-XXXX
	PurchaseOrderDate time.Time          `bson:"purchaseOrderDate" json:"purchaseOrderDate"`                             // วันที่ PO
	SupplierID        string             `bson:"supplierId" json:"supplierId"`                                   // รหัสซัพพลายเออร์
	SupplierName      string             `bson:"supplierName" json:"supplierName"`                               // ชื่อซัพพลายเออร์
	ContactName       *string            `bson:"contactName,omitempty" json:"contactName,omitempty"`             // ชื่อผู้ติดต่อ
	SupplierCode      *string            `bson:"supplierCode,omitempty" json:"supplierCode,omitempty"`           // รหัสซัพพลายเออร์
	TaxID             *string            `bson:"taxId,omitempty" json:"taxId,omitempty"`                         // เลขประจำตัวผู้เสียภาษี
	Address           *string            `bson:"address,omitempty" json:"address,omitempty"`                     // ที่อยู่
	Phone             *string            `bson:"phone,omitempty" json:"phone,omitempty"`                         // เบอร์โทรศัพท์
	Items             []PurchaseOrderItem `bson:"items" json:"items"`                                             // รายการสินค้า
	IsVAT             bool               `bson:"isVAT" json:"isVAT"`                                             // มี VAT หรือไม่
	VATType           string             `bson:"vatType" json:"vatType"`                                         // "exclusive" (VAT นอก) or "inclusive" (VAT ใน)
	ShippingCost      float64            `bson:"shippingCost" json:"shippingCost"`                               // ค่าขนส่ง
	Notes             *string            `bson:"notes,omitempty" json:"notes,omitempty"`                         // หมายเหตุ
	ValidUntil        *time.Time         `bson:"validUntil,omitempty" json:"validUntil,omitempty"`               // ราคาใช้ได้ถึง
	Status            string             `bson:"status" json:"status"`                                           // สถานะ (draft, sent, accepted, rejected, expired)
	PurchaseCode      *string            `bson:"purchaseCode,omitempty" json:"purchaseCode,omitempty"`                   // รหัสรายการซื้อที่สร้างจาก PO นี้
	BankAccountID     *string            `bson:"bankAccountId,omitempty" json:"bankAccountId,omitempty"`         // รหัสบัญชีธนาคาร
	BankName          *string            `bson:"bankName,omitempty" json:"bankName,omitempty"`                   // ชื่อธนาคาร
	BankAccountName   *string            `bson:"bankAccountName,omitempty" json:"bankAccountName,omitempty"`     // ชื่อบัญชี
	BankAccountNumber *string            `bson:"bankAccountNumber,omitempty" json:"bankAccountNumber,omitempty"` // เลขบัญชี
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// PurchaseOrderRequest represents the request body for creating/updating a purchase order
type PurchaseOrderRequest struct {
	PurchaseOrderDate CustomTime           `json:"purchaseOrderDate"`
	SupplierID        string               `json:"supplierId"`
	Items             []PurchaseOrderItem `json:"items"`
	IsVAT             bool                 `json:"isVAT"`
	VATType           string               `json:"vatType"` // "exclusive" or "inclusive"
	ShippingCost      float64              `json:"shippingCost"`
	Notes             *string              `json:"notes,omitempty"`
	ValidUntil        *CustomTime          `json:"validUntil,omitempty"`
	Status            string               `json:"status"`
	BankAccountID     *string              `json:"bankAccountId,omitempty"`
	BankName          *string              `json:"bankName,omitempty"`
	BankAccountName   *string              `json:"bankAccountName,omitempty"`
	BankAccountNumber *string              `json:"bankAccountNumber,omitempty"`
}

// ToPurchaseOrder converts PurchaseOrderRequest to PurchaseOrder
func (por *PurchaseOrderRequest) ToPurchaseOrder() *PurchaseOrder {
	now := utils.NowInThailand()
	po := &PurchaseOrder{
		PurchaseOrderDate: por.PurchaseOrderDate.Time,
		SupplierID:        por.SupplierID,
		Items:             por.Items,
		IsVAT:             por.IsVAT,
		VATType:           por.VATType,
		ShippingCost:      por.ShippingCost,
		Notes:             por.Notes,
		Status:            por.Status,
		BankAccountID:     por.BankAccountID,
		BankName:          por.BankName,
		BankAccountName:   por.BankAccountName,
		BankAccountNumber: por.BankAccountNumber,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if por.ValidUntil != nil {
		po.ValidUntil = &por.ValidUntil.Time
	}

	return po
}

// UpdateFromRequest updates PurchaseOrder from PurchaseOrderRequest
func (po *PurchaseOrder) UpdateFromRequest(por *PurchaseOrderRequest) {
	po.PurchaseOrderDate = por.PurchaseOrderDate.Time
	po.SupplierID = por.SupplierID
	po.Items = por.Items
	po.IsVAT = por.IsVAT
	po.VATType = por.VATType
	po.ShippingCost = por.ShippingCost
	po.Notes = por.Notes
	if por.ValidUntil != nil {
		po.ValidUntil = &por.ValidUntil.Time
	} else {
		po.ValidUntil = nil
	}
	po.Status = por.Status
	po.BankAccountID = por.BankAccountID
	po.BankName = por.BankName
	po.BankAccountName = por.BankAccountName
	po.BankAccountNumber = por.BankAccountNumber
	po.UpdatedAt = utils.NowInThailand()
}

// GeneratePurchaseOrderCode generates a new purchase order code in format PO-YYMM-XXXX
func GeneratePurchaseOrderCode(lastCode string) (string, error) {
	now := utils.NowInThailand()
	buddhistYear := now.Year() + 543 // Convert to Buddhist year
	month := int(now.Month())

	prefix := fmt.Sprintf("PO-%02d%02d-", buddhistYear%100, month) // YYMM

	if lastCode == "" {
		return prefix + "0001", nil
	}

	// Extract the numeric part (XXXX)
	var lastYear, lastMonth, lastSeq int
	_, err := fmt.Sscanf(lastCode, "PO-%02d%02d-%04d", &lastYear, &lastMonth, &lastSeq)
	if err != nil {
		return "", fmt.Errorf("invalid last purchase order code format: %w", err)
	}

	newSeq := lastSeq + 1
	return fmt.Sprintf("%s%04d", prefix, newSeq), nil
}

// CalculateGrandTotal calculates the grand total including VAT and shipping
func (po *PurchaseOrder) CalculateGrandTotal() float64 {
	totalBeforeVAT := 0.0
	for _, item := range po.Items {
		totalBeforeVAT += item.TotalPrice
	}

	var totalVAT float64
	var grandTotal float64
	if po.IsVAT {
		if po.VATType == "inclusive" {
			// VAT ใน: ราคารวม VAT แล้ว ต้องถอด VAT ออก
			// ราคาก่อน VAT = ราคารวม / 1.07
			// VAT = ราคารวม - ราคาก่อน VAT
			totalVAT = totalBeforeVAT - (totalBeforeVAT / 1.07)
			grandTotal = totalBeforeVAT + po.ShippingCost // ราคาที่กรอกคือราคารวม VAT แล้ว
		} else {
			// VAT นอก (exclusive): ราคา + VAT 7%
			totalVAT = totalBeforeVAT * 0.07
			grandTotal = totalBeforeVAT + totalVAT + po.ShippingCost
		}
	} else {
		grandTotal = totalBeforeVAT + po.ShippingCost
	}

	return grandTotal
}

// ToPurchaseRequest converts PurchaseOrder to PurchaseRequest for copying to purchase
func (po *PurchaseOrder) ToPurchaseRequest() *PurchaseRequest {
	// Convert PurchaseOrderItem to PurchaseItem
	purchaseItems := make([]PurchaseItem, len(po.Items))
	for i, item := range po.Items {
		purchaseItems[i] = PurchaseItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductCode: item.ProductCode,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			TotalPrice:  item.TotalPrice,
		}
	}

	poCode := po.PurchaseOrderCode
	return &PurchaseRequest{
		PurchaseDate:     utils.NowInThailand(), // Use current date for purchase
		SupplierID:       po.SupplierID,
		Items:            purchaseItems,
		IsVAT:            po.IsVAT,
		VATType:          po.VATType,
		ShippingCost:     po.ShippingCost,
		PurchaseOrderCode: &poCode, // Reference to original purchase order
		Payment: PaymentInfo{
			IsPaid: false, // Default to unpaid
		},
		Warehouse: WarehouseInfo{
			IsUpdated:      false,
			ActualShipping: po.ShippingCost,
			Items:          []WarehouseItem{}, // Empty warehouse items
		},
		Notes: po.Notes,
	}
}

