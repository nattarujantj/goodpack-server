package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Sale struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SaleCode          string             `bson:"saleCode" json:"saleCode"`
	QuotationCode     *string            `bson:"quotationCode,omitempty" json:"quotationCode,omitempty"`
	SaleDate          time.Time          `bson:"saleDate" json:"saleDate"`
	CustomerID        string             `bson:"customerId" json:"customerId"`
	CustomerName      string             `bson:"customerName" json:"customerName"`
	ContactName       *string            `bson:"contactName,omitempty" json:"contactName,omitempty"`
	CustomerCode      *string            `bson:"customerCode,omitempty" json:"customerCode,omitempty"`
	TaxID             *string            `bson:"taxId,omitempty" json:"taxId,omitempty"`
	Address           *string            `bson:"address,omitempty" json:"address,omitempty"`
	Phone             *string            `bson:"phone,omitempty" json:"phone,omitempty"`
	Items             []SaleItem         `bson:"items" json:"items"`
	IsVAT             bool               `bson:"isVAT" json:"isVAT"`
	ShippingCost      float64            `bson:"shippingCost" json:"shippingCost"`
	Payment           PaymentInfo        `bson:"payment" json:"payment"`
	Warehouse         WarehouseInfo      `bson:"warehouse" json:"warehouse"`
	Notes             *string            `bson:"notes,omitempty" json:"notes,omitempty"`
	BankAccountID     *string            `bson:"bankAccountId,omitempty" json:"bankAccountId,omitempty"`
	BankName          *string            `bson:"bankName,omitempty" json:"bankName,omitempty"`
	BankAccountName   *string            `bson:"bankAccountName,omitempty" json:"bankAccountName,omitempty"`
	BankAccountNumber *string            `bson:"bankAccountNumber,omitempty" json:"bankAccountNumber,omitempty"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type SaleItem struct {
	ProductID   string  `bson:"productId" json:"productId"`
	ProductName string  `bson:"productName" json:"productName"`
	ProductCode string  `bson:"productCode" json:"productCode"`
	Quantity    int     `bson:"quantity" json:"quantity"`
	UnitPrice   float64 `bson:"unitPrice" json:"unitPrice"`
	TotalPrice  float64 `bson:"totalPrice" json:"totalPrice"`
}

type SaleRequest struct {
	SaleDate          time.Time     `json:"saleDate"`
	CustomerID        string        `json:"customerId"`
	Items             []SaleItem    `json:"items"`
	IsVAT             bool          `json:"isVAT"`
	ShippingCost      float64       `json:"shippingCost"`
	Payment           PaymentInfo   `json:"payment"`
	Warehouse         WarehouseInfo `json:"warehouse"`
	Notes             *string       `json:"notes,omitempty"`
	QuotationCode     *string       `json:"quotationCode,omitempty"`
	BankAccountID     *string       `json:"bankAccountId,omitempty"`
	BankName          *string       `json:"bankName,omitempty"`
	BankAccountName   *string       `json:"bankAccountName,omitempty"`
	BankAccountNumber *string       `json:"bankAccountNumber,omitempty"`
}

func (sr *SaleRequest) ToSale() *Sale {
	now := time.Now()
	return &Sale{
		SaleDate:          sr.SaleDate,
		CustomerID:        sr.CustomerID,
		Items:             sr.Items,
		IsVAT:             sr.IsVAT,
		ShippingCost:      sr.ShippingCost,
		Payment:           sr.Payment,
		Warehouse:         sr.Warehouse,
		Notes:             sr.Notes,
		QuotationCode:     sr.QuotationCode,
		BankAccountID:     sr.BankAccountID,
		BankName:          sr.BankName,
		BankAccountName:   sr.BankAccountName,
		BankAccountNumber: sr.BankAccountNumber,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func (s *Sale) UpdateFromRequest(req *SaleRequest) {
	s.SaleDate = req.SaleDate
	s.CustomerID = req.CustomerID
	s.Items = req.Items
	s.IsVAT = req.IsVAT
	s.ShippingCost = req.ShippingCost
	s.Payment = req.Payment
	s.Warehouse = req.Warehouse
	s.Notes = req.Notes
	s.QuotationCode = req.QuotationCode
	s.BankAccountID = req.BankAccountID
	s.BankName = req.BankName
	s.BankAccountName = req.BankAccountName
	s.BankAccountNumber = req.BankAccountNumber
	s.UpdatedAt = time.Now()
}
