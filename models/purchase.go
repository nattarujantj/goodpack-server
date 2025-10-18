package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Purchase struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PurchaseCode string             `bson:"purchaseCode" json:"purchaseCode"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
	PurchaseDate time.Time          `bson:"purchaseDate" json:"purchaseDate"`
	CustomerID   string             `bson:"customerId" json:"customerId"`
	CustomerName string             `bson:"customerName" json:"customerName"`
	ContactName  *string            `bson:"contactName,omitempty" json:"contactName,omitempty"`
	CustomerCode *string            `bson:"customerCode,omitempty" json:"customerCode,omitempty"`
	TaxID        *string            `bson:"taxId,omitempty" json:"taxId,omitempty"`
	Address      *string            `bson:"address,omitempty" json:"address,omitempty"`
	Phone        *string            `bson:"phone,omitempty" json:"phone,omitempty"`
	Notes        *string            `bson:"notes,omitempty" json:"notes,omitempty"`
	Items        []PurchaseItem     `bson:"items" json:"items"`
	IsVAT        bool               `bson:"isVAT" json:"isVAT"`
	ShippingCost float64            `bson:"shippingCost" json:"shippingCost"`
	Payment      PaymentInfo        `bson:"payment" json:"payment"`
	Warehouse    WarehouseInfo      `bson:"warehouse" json:"warehouse"`
	TotalAmount  float64            `bson:"totalAmount" json:"totalAmount"`
	TotalVAT     float64            `bson:"totalVAT" json:"totalVAT"`
	GrandTotal   float64            `bson:"grandTotal" json:"grandTotal"`
}

type PurchaseItem struct {
	ProductID   string  `bson:"productId" json:"productId"`
	ProductName string  `bson:"productName" json:"productName"`
	ProductCode string  `bson:"productCode" json:"productCode"`
	Quantity    int     `bson:"quantity" json:"quantity"`
	UnitPrice   float64 `bson:"unitPrice" json:"unitPrice"`
	TotalPrice  float64 `bson:"totalPrice" json:"totalPrice"`
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
	PurchaseDate time.Time      `json:"purchaseDate" bson:"purchaseDate"`
	CustomerID   string         `json:"customerId" bson:"customerId"`
	Notes        *string        `json:"notes,omitempty" bson:"notes,omitempty"`
	Items        []PurchaseItem `json:"items" bson:"items"`
	IsVAT        bool           `json:"isVAT" bson:"isVAT"`
	ShippingCost float64        `json:"shippingCost" bson:"shippingCost"`
	Payment      PaymentInfo    `json:"payment" bson:"payment"`
	Warehouse    WarehouseInfo  `json:"warehouse" bson:"warehouse"`
}

func (pr *PurchaseRequest) ToPurchase() *Purchase {
	now := time.Now()

	// Calculate totals
	var totalAmount float64
	for _, item := range pr.Items {
		totalAmount += item.TotalPrice
	}

	var totalVAT float64
	if pr.IsVAT {
		totalVAT = totalAmount * 0.07 // 7% VAT
	}

	grandTotal := totalAmount + totalVAT

	return &Purchase{
		PurchaseCode: "", // Will be populated by handler
		CreatedAt:    now,
		UpdatedAt:    now,
		PurchaseDate: pr.PurchaseDate,
		CustomerID:   pr.CustomerID,
		CustomerName: "",  // Will be populated from customer data
		ContactName:  nil, // Will be populated from customer data
		CustomerCode: nil, // Will be populated from customer data
		TaxID:        nil, // Will be populated from customer data
		Address:      nil, // Will be populated from customer data
		Phone:        nil, // Will be populated from customer data
		Notes:        pr.Notes,
		Items:        pr.Items,
		IsVAT:        pr.IsVAT,
		ShippingCost: pr.ShippingCost,
		Payment:      pr.Payment,
		Warehouse:    pr.Warehouse,
		TotalAmount:  totalAmount,
		TotalVAT:     totalVAT,
		GrandTotal:   grandTotal,
	}
}

func (p *Purchase) UpdateFromRequest(pr *PurchaseRequest) {
	// Calculate totals
	var totalAmount float64
	for _, item := range pr.Items {
		totalAmount += item.TotalPrice
	}

	var totalVAT float64
	if pr.IsVAT {
		totalVAT = totalAmount * 0.07 // 7% VAT
	}

	grandTotal := totalAmount + totalVAT

	p.PurchaseDate = pr.PurchaseDate
	p.CustomerID = pr.CustomerID
	p.Notes = pr.Notes
	p.Items = pr.Items
	p.IsVAT = pr.IsVAT
	p.ShippingCost = pr.ShippingCost
	p.Payment = pr.Payment
	p.Warehouse = pr.Warehouse
	p.TotalAmount = totalAmount
	p.TotalVAT = totalVAT
	p.GrandTotal = grandTotal
	p.UpdatedAt = time.Now()
}
