package models

import (
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Purchase struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PurchaseCode  string             `bson:"purchaseCode" json:"purchaseCode"`
	InvoiceNumber *string            `bson:"invoiceNumber,omitempty" json:"invoiceNumber,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
	PurchaseDate  time.Time          `bson:"purchaseDate" json:"purchaseDate"`
	SupplierID    string             `bson:"supplierId" json:"supplierId"`
	SupplierName  string             `bson:"supplierName" json:"supplierName"`
	ContactName   *string            `bson:"contactName,omitempty" json:"contactName,omitempty"`
	SupplierCode  *string            `bson:"supplierCode,omitempty" json:"supplierCode,omitempty"`
	TaxID         *string            `bson:"taxId,omitempty" json:"taxId,omitempty"`
	Address       *string            `bson:"address,omitempty" json:"address,omitempty"`
	Phone         *string            `bson:"phone,omitempty" json:"phone,omitempty"`
	Notes         *string            `bson:"notes,omitempty" json:"notes,omitempty"`
	Items         []PurchaseItem     `bson:"items" json:"items"`
	IsVAT         bool               `bson:"isVAT" json:"isVAT"`
	VATType       string             `bson:"vatType" json:"vatType"` // "exclusive" (VAT นอก) or "inclusive" (VAT ใน)
	ShippingCost  float64            `bson:"shippingCost" json:"shippingCost"`
	Payment          PaymentInfo        `bson:"payment" json:"payment"`
	Warehouse        WarehouseInfo      `bson:"warehouse" json:"warehouse"`
	PurchaseOrderCode *string            `bson:"purchaseOrderCode,omitempty" json:"purchaseOrderCode,omitempty"`
	TotalAmount      float64            `bson:"totalAmount" json:"totalAmount"`
	TotalVAT         float64            `bson:"totalVAT" json:"totalVAT"`
	GrandTotal       float64            `bson:"grandTotal" json:"grandTotal"`
}

type PurchaseItem struct {
	ProductID        string   `bson:"productId" json:"productId"`
	ProductName      string   `bson:"productName" json:"productName"`
	ProductCode      string   `bson:"productCode" json:"productCode"`
	Quantity         int      `bson:"quantity" json:"quantity"`
	UnitPrice        float64  `bson:"unitPrice" json:"unitPrice"`
	PreformProductID *string  `bson:"preformProductId,omitempty" json:"preformProductId,omitempty"`
	PreformUnitPrice *float64 `bson:"preformUnitPrice,omitempty" json:"preformUnitPrice,omitempty"`
	TotalPrice       float64  `bson:"totalPrice" json:"totalPrice"`
}

type PaymentInfo struct {
	IsPaid          bool         `bson:"isPaid" json:"isPaid"`
	PaymentMethod   *string      `bson:"paymentMethod,omitempty" json:"paymentMethod,omitempty"`
	OurAccount      *string      `bson:"ourAccount,omitempty" json:"ourAccount,omitempty"`
	OurAccountInfo  *BankAccount `bson:"ourAccountInfo,omitempty" json:"ourAccountInfo,omitempty"`
	CustomerAccount *string      `bson:"customerAccount,omitempty" json:"customerAccount,omitempty"`
	PaymentDate     *time.Time   `bson:"paymentDate,omitempty" json:"paymentDate,omitempty"`
}

type BankAccount struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AccountNumber string `json:"accountNumber"`
	BankName      string `json:"bankName"`
	AccountType   string `json:"accountType"`
	IsActive      bool   `json:"isActive"`
}

type WarehouseInfo struct {
	IsUpdated      bool            `bson:"isUpdated" json:"isUpdated"`
	Notes          *string         `bson:"notes,omitempty" json:"notes,omitempty"`
	ActualShipping float64         `bson:"actualShipping" json:"actualShipping"`
	Items          []WarehouseItem `bson:"items" json:"items"`
}

type WarehouseItem struct {
	ProductID   string  `bson:"productId" json:"productId"`
	ProductName string  `bson:"productName" json:"productName"`
	Quantity    int     `bson:"quantity" json:"quantity"`
	Boxes       int     `bson:"boxes" json:"boxes"`
	Notes       *string `bson:"notes,omitempty" json:"notes,omitempty"`
}

type PurchaseRequest struct {
	PurchaseDate     time.Time      `json:"purchaseDate" bson:"purchaseDate"`
	SupplierID       string         `json:"supplierId" bson:"supplierId"`
	InvoiceNumber    *string        `json:"invoiceNumber,omitempty" bson:"invoiceNumber,omitempty"`
	Notes            *string        `json:"notes,omitempty" bson:"notes,omitempty"`
	Items            []PurchaseItem `json:"items" bson:"items"`
	IsVAT            bool           `json:"isVAT" bson:"isVAT"`
	VATType          string         `json:"vatType" bson:"vatType"` // "exclusive" or "inclusive"
	ShippingCost     float64        `json:"shippingCost" bson:"shippingCost"`
	Payment          PaymentInfo    `json:"payment" bson:"payment"`
	Warehouse        WarehouseInfo  `json:"warehouse" bson:"warehouse"`
	PurchaseOrderCode *string       `json:"purchaseOrderCode,omitempty" bson:"purchaseOrderCode,omitempty"`
}

func (pr *PurchaseRequest) ToPurchase() *Purchase {
	now := utils.NowInThailand()

	// Calculate totals
	var totalAmount float64
	for _, item := range pr.Items {
		totalAmount += item.TotalPrice
	}

	var totalVAT float64
	var grandTotal float64
	if pr.IsVAT {
		if pr.VATType == "inclusive" {
			// VAT ใน: ราคารวม VAT แล้ว ต้องถอด VAT ออก
			// ราคาก่อน VAT = ราคารวม / 1.07
			// VAT = ราคารวม - ราคาก่อน VAT
			totalVAT = totalAmount - (totalAmount / 1.07)
			grandTotal = totalAmount // ราคาที่กรอกคือราคารวม VAT แล้ว
		} else {
			// VAT นอก (exclusive): ราคา + VAT 7%
			totalVAT = totalAmount * 0.07
			grandTotal = totalAmount + totalVAT
		}
	} else {
		grandTotal = totalAmount
	}

	return &Purchase{
		PurchaseCode:  "", // Will be populated by handler
		InvoiceNumber: pr.InvoiceNumber,
		CreatedAt:     now,
		UpdatedAt:     now,
		PurchaseDate:  pr.PurchaseDate,
		SupplierID:    pr.SupplierID,
		SupplierName:  "", // Will be populated from supplier data
		ContactName:   nil, // Will be populated from supplier data
		SupplierCode:  nil, // Will be populated from supplier data
		TaxID:         nil, // Will be populated from supplier data
		Address:       nil, // Will be populated from supplier data
		Phone:         nil, // Will be populated from supplier data
		Notes:         pr.Notes,
		Items:         pr.Items,
		IsVAT:         pr.IsVAT,
		VATType:       pr.VATType,
		ShippingCost:  pr.ShippingCost,
		Payment:          pr.Payment,
		Warehouse:        pr.Warehouse,
		PurchaseOrderCode: pr.PurchaseOrderCode,
		TotalAmount:      totalAmount,
		TotalVAT:         totalVAT,
		GrandTotal:       grandTotal,
	}
}

func (p *Purchase) UpdateFromRequest(pr *PurchaseRequest) {
	// Calculate totals
	var totalAmount float64
	for _, item := range pr.Items {
		totalAmount += item.TotalPrice
	}

	var totalVAT float64
	var grandTotal float64
	if pr.IsVAT {
		if pr.VATType == "inclusive" {
			// VAT ใน: ราคารวม VAT แล้ว ต้องถอด VAT ออก
			totalVAT = totalAmount - (totalAmount / 1.07)
			grandTotal = totalAmount
		} else {
			// VAT นอก (exclusive): ราคา + VAT 7%
			totalVAT = totalAmount * 0.07
			grandTotal = totalAmount + totalVAT
		}
	} else {
		grandTotal = totalAmount
	}

	p.PurchaseDate = pr.PurchaseDate
	p.SupplierID = pr.SupplierID
	p.InvoiceNumber = pr.InvoiceNumber
	p.Notes = pr.Notes
	p.Items = pr.Items
	p.IsVAT = pr.IsVAT
	p.VATType = pr.VATType
	p.ShippingCost = pr.ShippingCost
	p.Payment = pr.Payment
	p.Warehouse = pr.Warehouse
	p.PurchaseOrderCode = pr.PurchaseOrderCode
	p.TotalAmount = totalAmount
	p.TotalVAT = totalVAT
	p.GrandTotal = grandTotal
	p.UpdatedAt = utils.NowInThailand()
}
