package models

import "time"

type ProductTransactionItem struct {
	ID           string    `json:"id"`
	DocumentCode string    `json:"documentCode"`
	Date         time.Time `json:"date"`
	PartnerName  string    `json:"partnerName"`
	Quantity     int       `json:"quantity"`
	UnitPrice    float64   `json:"unitPrice"`
	TotalPrice   float64   `json:"totalPrice"`
	TotalVAT     float64   `json:"totalVAT"`
	GrandTotal   float64   `json:"grandTotal"`
}

type ProductTransactionPage struct {
	Data       []ProductTransactionItem `json:"data"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	TotalPages int                      `json:"totalPages"`
}
