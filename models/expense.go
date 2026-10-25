package models

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Expense struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Category    string             `bson:"category" json:"category"`
	Description string             `bson:"description" json:"description"`
	Amount      float64            `bson:"amount" json:"amount"` // ยอดที่จ่ายจริง (รวม VAT ถ้ามี)
	// VAT คือภาษีซื้อที่รวมอยู่ใน Amount ตามใบกำกับภาษี (0 = ไม่มี VAT)
	VAT          float64             `bson:"vat" json:"vat"`
	Vendor       string              `bson:"vendor,omitempty" json:"vendor,omitempty"`
	TaxInvoiceNo string              `bson:"taxInvoiceNo,omitempty" json:"taxInvoiceNo,omitempty"`
	ExpenseDate  time.Time           `bson:"expenseDate" json:"expenseDate"`
	Notes        string              `bson:"notes,omitempty" json:"notes,omitempty"`
	Attachments  []ExpenseAttachment `bson:"attachments,omitempty" json:"attachments"`
	CreatedAt    time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time           `bson:"updatedAt" json:"updatedAt"`
}

// ExpenseAttachment represents an uploaded bill/receipt file (PDF or image)
// linked to a single expense record.
type ExpenseAttachment struct {
	ID         string    `bson:"id" json:"id"`
	FileName   string    `bson:"fileName" json:"fileName"`
	FileURL    string    `bson:"fileUrl" json:"fileUrl"`
	FileType   string    `bson:"fileType" json:"fileType"` // "pdf" or "image"
	Size       int64     `bson:"size" json:"size"`
	UploadedAt time.Time `bson:"uploadedAt" json:"uploadedAt"`
}

// AmountBeforeVAT คือยอดค่าใช้จ่ายก่อน VAT
func (e *Expense) AmountBeforeVAT() float64 {
	return e.Amount - e.VAT
}

type ExpenseRequest struct {
	Category     string  `json:"category"`
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	VAT          float64 `json:"vat"`
	Vendor       string  `json:"vendor,omitempty"`
	TaxInvoiceNo string  `json:"taxInvoiceNo,omitempty"`
	ExpenseDate  string  `json:"expenseDate"`
	Notes        string  `json:"notes,omitempty"`
}

// Validate checks the fields shared by create and update.
func (r *ExpenseRequest) Validate() error {
	if r.Category == "" {
		return errors.New("Category is required")
	}
	if r.Amount <= 0 {
		return errors.New("Amount must be greater than 0")
	}
	if r.VAT < 0 || r.VAT >= r.Amount {
		return errors.New("VAT must be at least 0 and less than the amount")
	}
	return nil
}

var DefaultExpenseCategories = []string{
	"ค่าน้ำมัน",
	"ค่าเช่า",
	"เงินเดือน",
	"ค่าน้ำ",
	"ค่าไฟ",
	"ค่าโทรศัพท์/อินเทอร์เน็ต",
	"ค่าขนส่ง",
	"ค่าวัสดุสิ้นเปลือง",
	"ค่าซ่อมบำรุง",
	"อื่นๆ",
}
