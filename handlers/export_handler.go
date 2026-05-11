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
	purchaseRepo    *repository.PurchaseRepository
	saleRepo        *repository.SaleRepository
	customerRepo    *repository.CustomerRepository
	expenseRepo     *repository.ExpenseRepository
	snapshotService *services.InventorySnapshotService
	excelService    *services.ExcelService
	emailService    *services.EmailService
}

func NewExportHandler(
	purchaseRepo *repository.PurchaseRepository,
	saleRepo *repository.SaleRepository,
	customerRepo *repository.CustomerRepository,
	expenseRepo *repository.ExpenseRepository,
	snapshotService *services.InventorySnapshotService,
) *ExportHandler {
	return &ExportHandler{
		purchaseRepo:    purchaseRepo,
		saleRepo:        saleRepo,
		customerRepo:    customerRepo,
		expenseRepo:     expenseRepo,
		snapshotService: snapshotService,
		excelService:    services.NewExcelService(),
		emailService:    services.NewEmailService(),
	}
}

// enrichSaleWithCustomerData enriches a sale with customer data
func (h *ExportHandler) enrichSaleWithCustomerData(sale *models.Sale) {
	if sale.CustomerID == "" {
		return
	}
	customer, err := h.customerRepo.GetByID(sale.CustomerID)
	if err == nil && customer != nil {
		sale.CustomerName = customer.CompanyName
		if sale.CustomerName == "" {
			sale.CustomerName = customer.ContactName
		}
		sale.TaxID = &customer.TaxID
	}
}

// enrichPurchaseWithSupplierData enriches a purchase with supplier data
func (h *ExportHandler) enrichPurchaseWithSupplierData(purchase *models.Purchase) {
	if purchase.SupplierID == "" {
		return
	}
}

type ExportEmailRequest struct {
	Month            int      `json:"month"`
	Year             int      `json:"year"`
	Emails           []string `json:"emails"`
	IncludeInventory bool     `json:"includeInventory"`
	InventoryMonth   int      `json:"inventoryMonth"` // 0 = same as Month
	InventoryYear    int      `json:"inventoryYear"`  // 0 = same as Year
}

type ExportEmailResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	PurchaseCount int    `json:"purchaseCount,omitempty"`
	SaleCount     int    `json:"saleCount,omitempty"`
	ExpenseCount  int    `json:"expenseCount,omitempty"`
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

	log.Printf("📤 Export request: Month=%d, Year=%d, Emails=%v, IncludeInventory=%v", req.Month, req.Year, req.Emails, req.IncludeInventory)

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

	// Filter purchases by date and enrich with customer data
	var purchases []models.Purchase
	for _, p := range allPurchases {
		if p.PurchaseDate.After(startDate.Add(-time.Second)) && p.PurchaseDate.Before(endDate.Add(time.Second)) {
			h.enrichPurchaseWithSupplierData(p)
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

	// Filter sales by date and enrich with customer data
	var sales []models.Sale
	for _, s := range allSales {
		if s.SaleDate.After(startDate.Add(-time.Second)) && s.SaleDate.Before(endDate.Add(time.Second)) {
			h.enrichSaleWithCustomerData(&s)
			sales = append(sales, s)
		}
	}

	// Fetch expenses for the month
	allExpenses, err := h.expenseRepo.GetAll(ctx)
	if err != nil {
		log.Printf("❌ Failed to fetch expenses: %v", err)
		sendErrorResponse(w, "Failed to fetch expenses")
		return
	}

	var expenses []models.Expense
	for _, e := range allExpenses {
		if e.ExpenseDate.After(startDate.Add(-time.Second)) && e.ExpenseDate.Before(endDate.Add(time.Second)) {
			expenses = append(expenses, *e)
		}
	}

	log.Printf("📊 Found %d purchases, %d sales, %d expenses for %d/%d", len(purchases), len(sales), len(expenses), req.Month, req.Year)

	// Optionally fetch inventory snapshot
	var snapshot *models.InventorySnapshot
	if req.IncludeInventory && h.snapshotService != nil {
		invMonth := req.Month
		invYear := req.Year
		if req.InventoryMonth > 0 {
			invMonth = req.InventoryMonth
		}
		if req.InventoryYear > 0 {
			invYear = req.InventoryYear
		}
		snap, err := h.snapshotService.GetSnapshot(ctx, invMonth, invYear)
		if err != nil {
			log.Printf("⚠️ Failed to fetch inventory snapshot: %v", err)
		} else {
			snapshot = snap
		}
	}

	// Generate Excel file
	excelBuffer, err := h.excelService.GeneratePurchaseSaleExcel(purchases, sales, expenses, req.Month, req.Year, snapshot)
	if err != nil {
		log.Printf("❌ Failed to generate Excel: %v", err)
		sendErrorResponse(w, "Failed to generate Excel file")
		return
	}

	// Build email body
	emailBody := h.emailService.BuildEmailBody(req.Month, req.Year, len(purchases), len(sales), len(expenses))

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
		ExpenseCount:  len(expenses),
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
