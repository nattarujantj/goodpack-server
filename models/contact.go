package models

// Contact represents a contact person for a customer or supplier
type Contact struct {
	Name      string `bson:"name" json:"name"`           // ชื่อผู้ติดต่อ
	Phone     string `bson:"phone" json:"phone"`         // เบอร์โทร
	IsDefault bool   `bson:"isDefault" json:"isDefault"` // ผู้ติดต่อหลัก
}

