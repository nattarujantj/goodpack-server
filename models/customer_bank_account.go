package models

// CustomerBankAccount represents a bank account for a customer or supplier
type CustomerBankAccount struct {
	BankName      string `bson:"bankName" json:"bankName"`           // ชื่อธนาคาร
	AccountName   string `bson:"accountName" json:"accountName"`     // ชื่อบัญชี
	AccountNumber string `bson:"accountNumber" json:"accountNumber"` // เลขบัญชี
	BranchName    string `bson:"branchName" json:"branchName"`       // สาขา
	IsDefault     bool   `bson:"isDefault" json:"isDefault"`         // บัญชีหลัก
}

