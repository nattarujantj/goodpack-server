package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Expense struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Category    string             `bson:"category" json:"category"`
	Description string             `bson:"description" json:"description"`
	Amount      float64            `bson:"amount" json:"amount"`
	ExpenseDate time.Time          `bson:"expenseDate" json:"expenseDate"`
	Notes       string             `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
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
