package models

type Account struct {
	ID            string `json:"id" bson:"id"`
	Name          string `json:"name" bson:"name"`
	AccountNumber string `json:"accountNumber" bson:"accountNumber"`
	BankName      string `json:"bankName" bson:"bankName"`
	AccountType   string `json:"accountType" bson:"accountType"`
	IsActive      bool   `json:"isActive" bson:"isActive"`
}

func (a *Account) GetDisplayName() string {
	return a.Name + " " + a.AccountNumber
}
