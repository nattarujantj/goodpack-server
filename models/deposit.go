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
	Amount      float64            `bson:"amount" json:"amount"`
	DepositDate time.Time          `bson:"depositDate" json:"depositDate"`
	// OurAccount คือบัญชีของเราที่รับ/จ่ายเงินมัดจำ (id จาก config)
	OurAccount *string   `bson:"ourAccount,omitempty" json:"ourAccount,omitempty"`
	Reference  string    `bson:"reference,omitempty" json:"reference,omitempty"` // เลขที่ PO/ใบเสร็จ/อ้างอิง
	Notes      string    `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt" json:"updatedAt"`

	// ค่าที่คำนวณจากรายการขาย/ซื้อที่หักมัดจำนี้ (ไม่เก็บใน DB)
	UsedAmount      float64        `bson:"-" json:"usedAmount"`
	RemainingAmount float64        `bson:"-" json:"remainingAmount"`
	Usages          []DepositUsage `bson:"-" json:"usages"`
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
type DepositApplication struct {
	DepositID   string  `bson:"depositId" json:"depositId"`
	DepositCode string  `bson:"depositCode" json:"depositCode"`
	Amount      float64 `bson:"amount" json:"amount"`
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
	d.UsedAmount = utils.RoundTo2(used)
	d.RemainingAmount = utils.RoundTo2(d.Amount - d.UsedAmount)
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
	DepositCode *string   `json:"depositCode,omitempty"`
	Type        string    `json:"type"`
	PartyID     string    `json:"partyId"`
	Amount      float64   `json:"amount"`
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

func (r *DepositRequest) ToDeposit() *Deposit {
	now := utils.NowInThailand()
	return &Deposit{
		Type:        r.Type,
		PartyID:     strings.TrimSpace(r.PartyID),
		Amount:      utils.RoundTo2(r.Amount),
		DepositDate: r.DepositDate,
		OurAccount:  r.OurAccount,
		Reference:   strings.TrimSpace(r.Reference),
		Notes:       r.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (d *Deposit) UpdateFromRequest(r *DepositRequest) {
	if r.DepositCode != nil && strings.TrimSpace(*r.DepositCode) != "" {
		d.DepositCode = strings.TrimSpace(*r.DepositCode)
	}
	d.Type = r.Type
	d.PartyID = strings.TrimSpace(r.PartyID)
	d.Amount = utils.RoundTo2(r.Amount)
	d.DepositDate = r.DepositDate
	d.OurAccount = r.OurAccount
	d.Reference = strings.TrimSpace(r.Reference)
	d.Notes = r.Notes
	d.UpdatedAt = utils.NowInThailand()
}
