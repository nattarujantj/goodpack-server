package models

import (
	"errors"
	"math"
	"strings"
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ประเภทเงินมัดจำ
const (
	// DepositTypeCustomer คือลูกค้ามาวางมัดจำกับเรา (ใช้หักในรายการขาย)
	DepositTypeCustomer = "customer"
	// DepositTypeSupplier คือเราไปวางมัดจำกับซัพพลายเออร์ (ใช้หักในรายการซื้อ)
	DepositTypeSupplier = "supplier"
)

// ประเภทเอกสารที่หักเงินมัดจำ
const (
	DepositDocSale     = "sale"
	DepositDocPurchase = "purchase"
)

// depositTolerance กันปัดเศษทศนิยมตอนเทียบยอดคงเหลือ
const depositTolerance = 0.005

// Deposit คือเงินมัดจำหนึ่งก้อน ซึ่งทยอยหักได้จากหลายรายการขาย/รายการซื้อ
// ของลูกค้า/ซัพพลายเออร์เจ้าเดียวกัน
type Deposit struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	DepositCode string             `bson:"depositCode" json:"depositCode"`
	Type        string             `bson:"type" json:"type"`       // "customer" หรือ "supplier"
	PartyID     string             `bson:"partyId" json:"partyId"` // customerId หรือ supplierId
	PartyName   string             `bson:"partyName" json:"partyName"`
	PartyCode   string             `bson:"partyCode,omitempty" json:"partyCode,omitempty"`
	// Amount คือยอดเงินที่รับ/จ่ายจริง (รวม VAT ถ้ามี)
	Amount float64 `bson:"amount" json:"amount"`
	// IsVAT = มัดจำนี้ออกใบกำกับภาษีแล้ว: ส่วนก่อน VAT ใช้หักก่อนคิด VAT ในรายการขาย/ซื้อที่เป็น VAT
	IsVAT           bool      `bson:"isVAT" json:"isVAT"`
	VatType         string    `bson:"vatType,omitempty" json:"vatType,omitempty"` // "exclusive" (VAT นอก) หรือ "inclusive" (VAT ใน) ตอนกรอก
	AmountBeforeVAT float64   `bson:"amountBeforeVAT" json:"amountBeforeVAT"`
	VATAmount       float64   `bson:"vatAmount" json:"vatAmount"`
	DepositDate     time.Time `bson:"depositDate" json:"depositDate"`
	// OurAccount คือบัญชีของเราที่รับ/จ่ายเงินมัดจำ (id จาก config)
	OurAccount *string   `bson:"ourAccount,omitempty" json:"ourAccount,omitempty"`
	Reference  string    `bson:"reference,omitempty" json:"reference,omitempty"` // เลขที่ PO/ใบเสร็จ/อ้างอิง
	Notes      string    `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt" json:"updatedAt"`

	// ค่าที่คำนวณจากรายการขาย/ซื้อที่หักมัดจำนี้ (ไม่เก็บใน DB)
	// หน่วยของ Deductible/Used/Remaining: มัดจำ VAT = ยอดก่อน VAT, มัดจำไม่มี VAT = ยอดเงินจริง
	DeductibleAmount float64        `bson:"-" json:"deductibleAmount"`
	UsedAmount       float64        `bson:"-" json:"usedAmount"`
	RemainingAmount  float64        `bson:"-" json:"remainingAmount"`
	Usages           []DepositUsage `bson:"-" json:"usages"`
}

// DepositUsage คือการหักเงินมัดจำหนึ่งครั้งในรายการขาย/ซื้อ
type DepositUsage struct {
	DocType string    `json:"docType"` // "sale" หรือ "purchase"
	DocID   string    `json:"docId"`
	DocCode string    `json:"docCode"`
	DocDate time.Time `json:"docDate"`
	Amount  float64   `json:"amount"`
}

// DepositApplication คือยอดมัดจำที่หักในรายการขาย/ซื้อหนึ่งรายการ (เก็บไว้ในรายการขาย/ซื้อ)
// BeforeVAT = true: เป็นมัดจำที่ออกใบกำกับภาษีแล้ว Amount คือยอดก่อน VAT และหักก่อนคิด VAT
// BeforeVAT = false: มัดจำไม่มี VAT หักออกจากยอดรวมหลัง VAT
type DepositApplication struct {
	DepositID   string  `bson:"depositId" json:"depositId"`
	DepositCode string  `bson:"depositCode" json:"depositCode"`
	Amount      float64 `bson:"amount" json:"amount"`
	BeforeVAT   bool    `bson:"beforeVAT" json:"beforeVAT"`
}

// SplitDepositApplications แยกยอดมัดจำที่หักก่อน VAT และหลัง VAT
func SplitDepositApplications(apps []DepositApplication) (beforeVAT, afterVAT float64) {
	for _, a := range apps {
		if a.BeforeVAT {
			beforeVAT += a.Amount
		} else {
			afterVAT += a.Amount
		}
	}
	return utils.RoundTo2(beforeVAT), utils.RoundTo2(afterVAT)
}

// SumDepositApplications รวมยอดมัดจำที่หักทั้งหมด
func SumDepositApplications(apps []DepositApplication) float64 {
	total := 0.0
	for _, a := range apps {
		total += a.Amount
	}
	return utils.RoundTo2(total)
}

// NormalizeDepositApplications รวมรายการซ้ำของมัดจำก้อนเดียวกัน ปัดทศนิยม 2 ตำแหน่ง
// และตัดรายการที่ยอด <= 0 ทิ้ง
func NormalizeDepositApplications(apps []DepositApplication) []DepositApplication {
	byID := map[string]int{}
	result := []DepositApplication{}
	for _, a := range apps {
		id := strings.TrimSpace(a.DepositID)
		if id == "" {
			continue
		}
		if idx, ok := byID[id]; ok {
			result[idx].Amount += a.Amount
			continue
		}
		byID[id] = len(result)
		result = append(result, DepositApplication{DepositID: id, Amount: a.Amount})
	}
	filtered := result[:0]
	for _, a := range result {
		a.Amount = utils.RoundTo2(a.Amount)
		if a.Amount > 0 {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// ApplyUsages ใส่ประวัติการหักมัดจำ และคำนวณยอดที่ใช้ไป/คงเหลือ
func (d *Deposit) ApplyUsages(usages []DepositUsage) {
	if usages == nil {
		usages = []DepositUsage{}
	}
	d.Usages = usages
	used := 0.0
	for _, u := range usages {
		used += u.Amount
	}
	d.DeductibleAmount = d.Deductible()
	d.UsedAmount = utils.RoundTo2(used)
	d.RemainingAmount = utils.RoundTo2(d.DeductibleAmount - d.UsedAmount)
}

// Deductible คือยอดที่ใช้หักได้ทั้งหมด: มัดจำ VAT ใช้ยอดก่อน VAT (เพราะ VAT ส่วนนี้ออกใบกำกับไปแล้ว)
func (d *Deposit) Deductible() float64 {
	if d.IsVAT {
		return d.AmountBeforeVAT
	}
	return d.Amount
}

// CalculateDepositVAT แยกยอดก่อน VAT / VAT / ยอดรวม จากยอดที่กรอก
// exclusive: ยอดที่กรอกคือยอดก่อน VAT, inclusive: ยอดที่กรอกรวม VAT แล้ว
func CalculateDepositVAT(entered float64, isVAT bool, vatType string) (beforeVAT, vat, total float64) {
	entered = utils.RoundTo2(entered)
	if !isVAT {
		return entered, 0, entered
	}
	if vatType == "inclusive" {
		total = entered
		beforeVAT = utils.RoundTo2(total / 1.07)
		return beforeVAT, utils.RoundTo2(total - beforeVAT), total
	}
	beforeVAT = entered
	vat = utils.RoundTo2(beforeVAT * 0.07)
	return beforeVAT, vat, utils.RoundTo2(beforeVAT + vat)
}

// CanCover บอกว่ายอดคงเหลือ (ไม่นับเอกสารที่กำลังแก้ไข) พอให้หักตามยอดที่ขอหรือไม่
func (d *Deposit) CanCover(amount float64) bool {
	return amount <= d.RemainingAmount+depositTolerance
}

// DocTypeForDeposit คือประเภทเอกสารที่ใช้หักมัดจำแต่ละประเภท
func DocTypeForDeposit(depositType string) string {
	if depositType == DepositTypeSupplier {
		return DepositDocPurchase
	}
	return DepositDocSale
}

type DepositRequest struct {
	DepositCode *string `json:"depositCode,omitempty"`
	Type        string  `json:"type"`
	PartyID     string  `json:"partyId"`
	// Amount คือยอดที่กรอก: ไม่มี VAT / VAT นอก = ยอดก่อน VAT, VAT ใน = ยอดรวม VAT
	Amount      float64   `json:"amount"`
	IsVAT       bool      `json:"isVAT"`
	VatType     string    `json:"vatType"`
	DepositDate time.Time `json:"depositDate"`
	OurAccount  *string   `json:"ourAccount,omitempty"`
	Reference   string    `json:"reference,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

func (r *DepositRequest) Validate() error {
	if r.Type != DepositTypeCustomer && r.Type != DepositTypeSupplier {
		return errors.New("type must be 'customer' or 'supplier'")
	}
	if strings.TrimSpace(r.PartyID) == "" {
		if r.Type == DepositTypeCustomer {
			return errors.New("กรุณาเลือกลูกค้า")
		}
		return errors.New("กรุณาเลือกซัพพลายเออร์")
	}
	if r.Amount <= 0 || math.IsNaN(r.Amount) || math.IsInf(r.Amount, 0) {
		return errors.New("ยอดเงินมัดจำต้องมากกว่า 0")
	}
	if r.DepositDate.IsZero() {
		return errors.New("กรุณาระบุวันที่วางมัดจำ")
	}
	return nil
}

func (r *DepositRequest) vatFields() (vatType string, beforeVAT, vat, total float64) {
	vatType = ""
	if r.IsVAT {
		vatType = "exclusive"
		if r.VatType == "inclusive" {
			vatType = "inclusive"
		}
	}
	beforeVAT, vat, total = CalculateDepositVAT(r.Amount, r.IsVAT, vatType)
	return
}

func (r *DepositRequest) ToDeposit() *Deposit {
	now := utils.NowInThailand()
	vatType, beforeVAT, vat, total := r.vatFields()
	return &Deposit{
		Type:            r.Type,
		PartyID:         strings.TrimSpace(r.PartyID),
		Amount:          total,
		IsVAT:           r.IsVAT,
		VatType:         vatType,
		AmountBeforeVAT: beforeVAT,
		VATAmount:       vat,
		DepositDate:     r.DepositDate,
		OurAccount:      r.OurAccount,
		Reference:       strings.TrimSpace(r.Reference),
		Notes:           r.Notes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (d *Deposit) UpdateFromRequest(r *DepositRequest) {
	if r.DepositCode != nil && strings.TrimSpace(*r.DepositCode) != "" {
		d.DepositCode = strings.TrimSpace(*r.DepositCode)
	}
	d.Type = r.Type
	d.PartyID = strings.TrimSpace(r.PartyID)
	vatType, beforeVAT, vat, total := r.vatFields()
	d.Amount = total
	d.IsVAT = r.IsVAT
	d.VatType = vatType
	d.AmountBeforeVAT = beforeVAT
	d.VATAmount = vat
	d.DepositDate = r.DepositDate
	d.OurAccount = r.OurAccount
	d.Reference = strings.TrimSpace(r.Reference)
	d.Notes = r.Notes
	d.UpdatedAt = utils.NowInThailand()
}
