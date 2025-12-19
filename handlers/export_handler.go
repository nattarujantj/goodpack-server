package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/services"
)

type ExportHandler struct {
	purchaseRepo *repository.PurchaseRepository
	saleRepo     *repository.SaleRepository
	excelService *services.ExcelService
	emailService *services.EmailService
}

func NewExportHandler(purchaseRepo *repository.PurchaseRepository, saleRepo *repository.SaleRepository) *ExportHandler {
	return &ExportHandler{
		purchaseRepo: purchaseRepo,
		saleRepo:     saleRepo,
		excelService: services.NewExcelService(),
		emailService: services.NewEmailService(),
	}
}

type ExportEmailRequest struct {
	Month  int      `json:"month"`
	Year   int      `json:"year"`
	Emails []string `json:"emails"`
}

type ExportEmailResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	PurchaseCount int    `json:"purchaseCount,omitempty"`
	SaleCount     int    `json:"saleCount,omitempty"`
}

// ExportAndSendEmail handles the export request
func (h *ExportHandler) ExportAndSendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExportEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body")
		return
	}

	// Validate request
	if req.Month < 1 || req.Month > 12 {
		sendErrorResponse(w, "Invalid month (1-12)")
		return
	}
	if req.Year < 2020 || req.Year > 2100 {
		sendErrorResponse(w, "Invalid year")
		return
	}
	if len(req.Emails) == 0 {
		sendErrorResponse(w, "At least one email is required")
		return
	}

	log.Printf("📤 Export request: Month=%d, Year=%d, Emails=%v", req.Month, req.Year, req.Emails)

	ctx := r.Context()

	// Calculate date range for the month
	startDate := time.Date(req.Year, time.Month(req.Month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second) // Last second of the month

	// Fetch purchases for the month
	allPurchases, err := h.purchaseRepo.GetAll(ctx)
	if err != nil {
		log.Printf("❌ Failed to fetch purchases: %v", err)
		sendErrorResponse(w, "Failed to fetch purchases")
		return
	}

	// Filter purchases by date
	var purchases []models.Purchase
	for _, p := range allPurchases {
		if p.PurchaseDate.After(startDate.Add(-time.Second)) && p.PurchaseDate.Before(endDate.Add(time.Second)) {
			purchases = append(purchases, *p)
		}
	}

	// Fetch sales for the month
	allSales, err := h.saleRepo.GetAll(ctx)
	if err != nil {
		log.Printf("❌ Failed to fetch sales: %v", err)
		sendErrorResponse(w, "Failed to fetch sales")
		return
	}

	// Filter sales by date
	var sales []models.Sale
	for _, s := range allSales {
		if s.SaleDate.After(startDate.Add(-time.Second)) && s.SaleDate.Before(endDate.Add(time.Second)) {
			sales = append(sales, s)
		}
	}

	log.Printf("📊 Found %d purchases and %d sales for %d/%d", len(purchases), len(sales), req.Month, req.Year)

	// Generate Excel file
	excelBuffer, err := h.excelService.GeneratePurchaseSaleExcel(purchases, sales, req.Month, req.Year)
	if err != nil {
		log.Printf("❌ Failed to generate Excel: %v", err)
		sendErrorResponse(w, "Failed to generate Excel file")
		return
	}

	// Build email body
	emailBody := h.emailService.BuildEmailBody(req.Month, req.Year, len(purchases), len(sales))

	// Generate filename
	thaiMonth := services.GetThaiMonthName(req.Month)
	buddhistYear := req.Year + 543
	filename := fmt.Sprintf("GoodPack_Report_%s_%d.xlsx", thaiMonth, buddhistYear)

	// Email subject
	subject := fmt.Sprintf("📦 GoodPack รายงานซื้อ-ขาย %s %d", thaiMonth, buddhistYear)

	// Send email
	err = h.emailService.SendEmailWithAttachment(req.Emails, subject, emailBody, excelBuffer, filename)
	if err != nil {
		log.Printf("❌ Failed to send email: %v", err)
		sendErrorResponse(w, fmt.Sprintf("Failed to send email: %v", err))
		return
	}

	log.Printf("✅ Export sent successfully to %v", req.Emails)

	// Send success response
	response := ExportEmailResponse{
		Success:       true,
		Message:       fmt.Sprintf("ส่งรายงาน %s %d ไปยัง %d email สำเร็จ!", thaiMonth, buddhistYear, len(req.Emails)),
		PurchaseCount: len(purchases),
		SaleCount:     len(sales),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendErrorResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ExportEmailResponse{
		Success: false,
		Message: message,
	})
}

