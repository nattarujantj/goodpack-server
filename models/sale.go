package models

import (
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Sale struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SaleCode      string             `bson:"saleCode" json:"saleCode"`
	QuotationCode *string            `bson:"quotationCode,omitempty" json:"quotationCode,omitempty"`
	SaleDate      time.Time          `bson:"saleDate" json:"saleDate"`
	CustomerID    string             `bson:"customerId" json:"customerId"`
	CustomerName  string             `bson:"customerName" json:"customerName"`
	ContactName   *string            `bson:"contactName,omitempty" json:"contactName,omitempty"`
	CustomerCode  *string            `bson:"customerCode,omitempty" json:"customerCode,omitempty"`
	TaxID         *string            `bson:"taxId,omitempty" json:"taxId,omitempty"`
	Address       *string            `bson:"address,omitempty" json:"address,omitempty"`
	Phone         *string            `bson:"phone,omitempty" json:"phone,omitempty"`
	Items         []SaleItem         `bson:"items" json:"items"`
	IsVAT         bool               `bson:"isVAT" json:"isVAT"`
	VatType       string             `bson:"vatType" json:"vatType"` // "exclusive" (VAT นอก) or "inclusive" (VAT ใน)
	ShippingCost  float64            `bson:"shippingCost" json:"shippingCost"`
	Deposit       float64            `bson:"deposit" json:"deposit"` // เงินมัดจำ (หักออกจากยอดที่ต้องชำระหลัง VAT)
	// DepositApplications คือเงินมัดจำของลูกค้าที่หักในรายการนี้ (ถ้ามี Deposit = ผลรวมของรายการนี้)
	DepositApplications []DepositApplication `bson:"depositApplications,omitempty" json:"depositApplications"`
	// DepositBeforeVAT คือมัดจำที่ออกใบกำกับภาษีแล้ว หักออกจากยอดสินค้าก่อนคิด VAT
	DepositBeforeVAT  float64       `bson:"depositBeforeVAT" json:"depositBeforeVAT"`
	Payment           PaymentInfo   `bson:"payment" json:"payment"`
	Warehouse         WarehouseInfo `bson:"warehouse" json:"warehouse"`
	Notes             *string       `bson:"notes,omitempty" json:"notes,omitempty"`
	BankAccountID     *string       `bson:"bankAccountId,omitempty" json:"bankAccountId,omitempty"`
	BankName          *string       `bson:"bankName,omitempty" json:"bankName,omitempty"`
	BankAccountName   *string       `bson:"bankAccountName,omitempty" json:"bankAccountName,omitempty"`
	BankAccountNumber *string       `bson:"bankAccountNumber,omitempty" json:"bankAccountNumber,omitempty"`
	CreatedAt         time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time     `bson:"updatedAt" json:"updatedAt"`
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
	SaleCode     *string    `json:"saleCode,omitempty"`
	SaleDate     time.Time  `json:"saleDate"`
	CustomerID   string     `json:"customerId"`
	Items        []SaleItem `json:"items"`
	IsVAT        bool       `json:"isVAT"`
	VatType      string     `json:"vatType"` // "exclusive" or "inclusive"
	ShippingCost float64    `json:"shippingCost"`
	Deposit      float64    `json:"deposit"`
	// DepositApplications คือเงินมัดจำที่เลือกหัก ถ้าส่งมาจะใช้ผลรวมแทน Deposit
	DepositApplications []DepositApplication `json:"depositApplications"`
	DepositBeforeVAT    float64              `json:"-"` // คำนวณโดย server จาก DepositApplications
	Payment             PaymentInfo          `json:"payment"`
	Warehouse           WarehouseInfo        `json:"warehouse"`
	Notes               *string              `json:"notes,omitempty"`
	QuotationCode       *string              `json:"quotationCode,omitempty"`
	BankAccountID       *string              `json:"bankAccountId,omitempty"`
	BankName            *string              `json:"bankName,omitempty"`
	BankAccountName     *string              `json:"bankAccountName,omitempty"`
	BankAccountNumber   *string              `json:"bankAccountNumber,omitempty"`
}

func (sr *SaleRequest) ToSale() *Sale {
	now := utils.NowInThailand()
	vatType := sr.VatType
	if vatType == "" {
		vatType = "exclusive" // Default to VAT นอก
	}
	return &Sale{
		SaleDate:            sr.SaleDate,
		CustomerID:          sr.CustomerID,
		Items:               sr.Items,
		IsVAT:               sr.IsVAT,
		VatType:             vatType,
		ShippingCost:        sr.ShippingCost,
		Deposit:             sr.Deposit,
		DepositApplications: sr.DepositApplications,
		DepositBeforeVAT:    sr.DepositBeforeVAT,
		Payment:             sr.Payment,
		Warehouse:           sr.Warehouse,
		Notes:               sr.Notes,
		QuotationCode:       sr.QuotationCode,
		BankAccountID:       sr.BankAccountID,
		BankName:            sr.BankName,
		BankAccountName:     sr.BankAccountName,
		BankAccountNumber:   sr.BankAccountNumber,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func (s *Sale) UpdateFromRequest(req *SaleRequest) {
	vatType := req.VatType
	if vatType == "" {
		vatType = "exclusive" // Default to VAT นอก
	}
	if req.SaleCode != nil && *req.SaleCode != "" {
		s.SaleCode = *req.SaleCode
	}
	s.SaleDate = req.SaleDate
	s.CustomerID = req.CustomerID
	s.Items = req.Items
	s.IsVAT = req.IsVAT
	s.VatType = vatType
	s.ShippingCost = req.ShippingCost
	s.Deposit = req.Deposit
	s.DepositApplications = req.DepositApplications
	s.DepositBeforeVAT = req.DepositBeforeVAT
	s.Payment = req.Payment
	s.Warehouse = req.Warehouse
	s.Notes = req.Notes
	s.QuotationCode = req.QuotationCode
	s.BankAccountID = req.BankAccountID
	s.BankName = req.BankName
	s.BankAccountName = req.BankAccountName
	s.BankAccountNumber = req.BankAccountNumber
	s.UpdatedAt = utils.NowInThailand()
}

// SaleTotals คือยอดของใบขาย (ไม่รวมค่าส่ง)
type SaleTotals struct {
	ItemsBeforeVAT float64 // ยอดสินค้าก่อน VAT
	NetBeforeVAT   float64 // ยอดก่อน VAT หลังหักมัดจำที่ออกใบกำกับแล้ว (ฐานภาษีของใบนี้)
	VAT            float64 // VAT ของใบนี้
	AfterVAT       float64 // NetBeforeVAT + VAT
}

// Totals คำนวณยอดของใบขาย สูตรเดียวกับหน้าเว็บ
// มัดจำที่ออกใบกำกับภาษีแล้ว (DepositBeforeVAT) หักก่อนคิด VAT
func (s *Sale) Totals() SaleTotals {
	items := 0.0
	for _, it := range s.Items {
		items += it.TotalPrice
	}
	if !s.IsVAT {
		return SaleTotals{ItemsBeforeVAT: items, NetBeforeVAT: items, AfterVAT: items}
	}
	var t SaleTotals
	if s.VatType == "inclusive" {
		t.ItemsBeforeVAT = utils.RoundTo2(items / 1.07)
		if s.DepositBeforeVAT <= 0 {
			t.NetBeforeVAT = t.ItemsBeforeVAT
			t.VAT = utils.RoundTo2(items - t.ItemsBeforeVAT)
			t.AfterVAT = items
			return t
		}
	} else {
		t.ItemsBeforeVAT = items
	}
	t.NetBeforeVAT = utils.RoundTo2(t.ItemsBeforeVAT - s.DepositBeforeVAT)
	t.VAT = utils.RoundTo2(t.NetBeforeVAT * 0.07)
	t.AfterVAT = utils.RoundTo2(t.NetBeforeVAT + t.VAT)
	return t
}
