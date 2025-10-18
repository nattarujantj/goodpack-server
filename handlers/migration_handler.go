package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type MigrationHandler struct {
	customerRepo *repository.CustomerRepository
}

func NewMigrationHandler(customerRepo *repository.CustomerRepository) *MigrationHandler {
	return &MigrationHandler{
		customerRepo: customerRepo,
	}
}

// CustomerCSVRow represents a row in the customer CSV file
type CustomerCSVRow struct {
	CustomerCode  string `csv:"customerCode"`
	CompanyName   string `csv:"companyName"`
	ContactName   string `csv:"contactName"`
	TaxID         string `csv:"taxId"`
	Phone         string `csv:"phone"`
	Address       string `csv:"address"`
	ContactMethod string `csv:"contactMethod"`
}

// MigrationResult represents the result of migration
type MigrationResult struct {
	TotalRows   int       `json:"totalRows"`
	SuccessRows int       `json:"successRows"`
	FailedRows  int       `json:"failedRows"`
	Errors      []string  `json:"errors"`
	ProcessedAt time.Time `json:"processedAt"`
}

// MigrateCustomersFromCSV handles CSV file upload and migration
func (h *MigrationHandler) MigrateCustomersFromCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max file size
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, _, err := r.FormFile("csvFile")
	if err != nil {
		http.Error(w, "No CSV file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse CSV
	result, err := h.parseAndMigrateCustomerCSV(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process CSV: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// parseAndMigrateCustomerCSV parses CSV file and migrates data to database
func (h *MigrationHandler) parseAndMigrateCustomerCSV(file io.Reader) (*MigrationResult, error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must have at least a header row and one data row")
	}

	// Get header row
	headers := records[0]
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	// Validate required headers
	requiredHeaders := []string{"companyname", "contactname"}
	optionalHeaders := []string{"customercode", "taxid", "phone", "address", "contactmethod"}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			return nil, fmt.Errorf("missing required header: %s", required)
		}
	}

	// Log available optional headers
	for _, optional := range optionalHeaders {
		if _, exists := headerMap[optional]; exists {
			fmt.Printf("Found optional header: %s\n", optional)
		}
	}

	result := &MigrationResult{
		TotalRows:   len(records) - 1, // Exclude header row
		SuccessRows: 0,
		FailedRows:  0,
		Errors:      []string{},
		ProcessedAt: time.Now(),
	}

	// Process data rows
	for i, record := range records[1:] {
		rowNum := i + 2 // +2 because we start from row 2 (after header)

		// Create customer from CSV row
		customer := &models.Customer{
			CustomerCode:  h.getFieldValue(record, headerMap, "customercode"),
			CompanyName:   h.getFieldValue(record, headerMap, "companyname"),
			ContactName:   h.getFieldValue(record, headerMap, "contactname"),
			TaxID:         h.getFieldValue(record, headerMap, "taxid"),
			Phone:         h.getFieldValue(record, headerMap, "phone"),
			Address:       h.getFieldValue(record, headerMap, "address"),
			ContactMethod: h.getFieldValue(record, headerMap, "contactmethod"),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Validate required fields
		if customer.CompanyName == "" {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Company name is required", rowNum))
			continue
		}

		if customer.ContactName == "" {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Contact name is required", rowNum))
			continue
		}

		// Handle customer code
		if customer.CustomerCode == "" {
			// Generate customer code if not provided
			customerCode, err := h.customerRepo.GenerateCustomerCode()
			if err != nil {
				result.FailedRows++
				result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to generate customer code - %v", rowNum, err))
				continue
			}
			customer.CustomerCode = customerCode
		} else {
			// Check if customer code already exists
			existingCustomer, err := h.customerRepo.GetByCustomerCode(customer.CustomerCode)
			if err == nil && existingCustomer != nil {
				result.FailedRows++
				result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Customer code '%s' already exists", rowNum, customer.CustomerCode))
				continue
			}
		}

		// Save to database
		err := h.customerRepo.Create(customer)
		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to save customer - %v", rowNum, err))
			continue
		}

		result.SuccessRows++
	}

	return result, nil
}

// getFieldValue safely gets field value from CSV record
func (h *MigrationHandler) getFieldValue(record []string, headerMap map[string]int, fieldName string) string {
	if index, exists := headerMap[fieldName]; exists && index < len(record) {
		return strings.TrimSpace(record[index])
	}
	return ""
}

// GetCustomerCSVTemplate returns a CSV template for customer data
func (h *MigrationHandler) GetCustomerCSVTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create CSV template
	template := "customerCode,companyName,contactName,taxId,phone,address,contactMethod\n"
	template += "C-0001,บริษัทตัวอย่าง จำกัด,นายสมชาย ใจดี,1234567890123,02-123-4567,123 ถนนสุขุมวิท กรุงเทพฯ 10110,email\n"
	template += ",บริษัททดสอบ จำกัด,นางสมหญิง รักดี,9876543210987,02-987-6543,456 ถนนรัชดาภิเษก กรุงเทพฯ 10400,phone\n"
	template += "C-0003,บริษัทสินค้าดี จำกัด,นายวิชัย เก่งมาก,1111111111111,02-111-2222,789 ถนนพหลโยธิน กรุงเทพฯ 10900,line\n"

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=customer_template.csv")
	w.Write([]byte(template))
}

// GetMigrationStatus returns the status of recent migrations
func (h *MigrationHandler) GetMigrationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get total customer count
	customers, err := h.customerRepo.GetAll()
	if err != nil {
		http.Error(w, "Failed to get customer count", http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"totalCustomers": len(customers),
		"lastChecked":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
