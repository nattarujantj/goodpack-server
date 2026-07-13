package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Expense struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Category    string              `bson:"category" json:"category"`
	Description string              `bson:"description" json:"description"`
	Amount      float64             `bson:"amount" json:"amount"`
	ExpenseDate time.Time           `bson:"expenseDate" json:"expenseDate"`
	Notes       string              `bson:"notes,omitempty" json:"notes,omitempty"`
	Attachments []ExpenseAttachment `bson:"attachments,omitempty" json:"attachments"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time           `bson:"updatedAt" json:"updatedAt"`
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

type ExpenseRequest struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	ExpenseDate string  `json:"expenseDate"`
	Notes       string  `json:"notes,omitempty"`
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
