package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type MigrationHandler struct {
	customerRepo *repository.CustomerRepository
	productRepo  *repository.ProductRepository
}

func NewMigrationHandler(customerRepo *repository.CustomerRepository, productRepo *repository.ProductRepository) *MigrationHandler {
	return &MigrationHandler{
		customerRepo: customerRepo,
		productRepo:  productRepo,
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

// ProductCSVRow represents a row in the product CSV file
type ProductCSVRow struct {
	SKUID               string `csv:"skuId"`
	Name                string `csv:"name"`
	Description         string `csv:"description"`
	Color               string `csv:"color"`
	Size                string `csv:"size"`
	Category            string `csv:"category"`
	PurchasePriceVAT    string `csv:"purchasePriceVAT"`
	PurchasePriceNonVAT string `csv:"purchasePriceNonVAT"`
	SalePriceVAT        string `csv:"salePriceVAT"`
	SalePriceNonVAT     string `csv:"salePriceNonVAT"`
	StockVAT            string `csv:"stockVAT"`
	StockNonVAT         string `csv:"stockNonVAT"`
	ActualStock         string `csv:"actualStock"`
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

// MigrateProductsFromCSV handles CSV file upload and migration for products
func (h *MigrationHandler) MigrateProductsFromCSV(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.parseAndMigrateProductCSV(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process CSV: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// parseAndMigrateProductCSV parses CSV file and migrates product data to database
func (h *MigrationHandler) parseAndMigrateProductCSV(file io.Reader) (*MigrationResult, error) {
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
	requiredHeaders := []string{"name", "category"}
	optionalHeaders := []string{"skuid", "description", "color", "size", "purchasepricevat", "purchasepricenonvat", "salepricevat", "salepricenonvat", "stockvat", "stocknonvat", "actualstock"}

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

		// Create product from CSV row
		product := &models.Product{
			SKUID:       h.getFieldValue(record, headerMap, "skuid"),
			Name:        h.getFieldValue(record, headerMap, "name"),
			Description: h.getFieldValue(record, headerMap, "description"),
			Color:       h.getFieldValue(record, headerMap, "color"),
			Size:        h.getFieldValue(record, headerMap, "size"),
			Category:    h.getFieldValue(record, headerMap, "category"),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Validate required fields
		if product.Name == "" {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Product name is required", rowNum))
			continue
		}

		if product.Category == "" {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Category is required", rowNum))
			continue
		}

		// Parse prices
		product.Price = h.parseProductPrices(record, headerMap)

		// Parse stock
		product.Stock = h.parseProductStock(record, headerMap)
		// Handle SKU ID
		if product.SKUID != "" {
			// Check if SKU ID already exists
			existingProduct, err := h.productRepo.GetBySKUID(context.Background(), product.SKUID)
			if err == nil && existingProduct != nil {
				result.FailedRows++
				result.Errors = append(result.Errors, fmt.Sprintf("Row %d: SKU ID '%s' already exists", rowNum, product.SKUID))
				continue
			}
		}
		// If SKUID is empty, it will be generated by the repository

		// Generate Product Code
		product.Code = h.generateProductCode(product.Category, product.Size, product.Color)

		// Save to database
		err := h.productRepo.Create(context.Background(), product)
		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to save product - %v", rowNum, err))
			continue
		}

		result.SuccessRows++
	}

	return result, nil
}

// parseProductPrices parses price information from CSV row
func (h *MigrationHandler) parseProductPrices(record []string, headerMap map[string]int) models.Price {
	price := models.Price{
		PurchaseVAT:    models.PriceInfo{},
		PurchaseNonVAT: models.PriceInfo{},
		SaleVAT:        models.PriceInfo{},
		SaleNonVAT:     models.PriceInfo{},
		SalesTiers:     []models.TierPrice{},
	}

	// Parse purchase prices
	if purchaseVAT := h.getFieldValue(record, headerMap, "purchasepricevat"); purchaseVAT != "" {
		if val, err := h.parseFloat(purchaseVAT); err == nil {
			price.PurchaseVAT.Latest = val
		}
	}

	if purchaseNonVAT := h.getFieldValue(record, headerMap, "purchasepricenonvat"); purchaseNonVAT != "" {
		if val, err := h.parseFloat(purchaseNonVAT); err == nil {
			price.PurchaseNonVAT.Latest = val
		}
	}

	// Parse sale prices
	if saleVAT := h.getFieldValue(record, headerMap, "salepricevat"); saleVAT != "" {
		if val, err := h.parseFloat(saleVAT); err == nil {
			price.SaleVAT.Latest = val
		}
	}

	if saleNonVAT := h.getFieldValue(record, headerMap, "salepricenonvat"); saleNonVAT != "" {
		if val, err := h.parseFloat(saleNonVAT); err == nil {
			price.SaleNonVAT.Latest = val
		}
	}

	return price
}

// parseProductStock parses stock information from CSV row
func (h *MigrationHandler) parseProductStock(record []string, headerMap map[string]int) models.Stock {
	stock := models.Stock{
		VAT:         models.StockInfo{},
		NonVAT:      models.StockInfo{},
		ActualStock: 0,
	}

	// Parse VAT stock
	if stockVAT := h.getFieldValue(record, headerMap, "stockvat"); stockVAT != "" {
		if val, err := h.parseInt(stockVAT); err == nil {
			stock.VAT.Remaining = val
		}
	}

	// Parse Non-VAT stock
	if stockNonVAT := h.getFieldValue(record, headerMap, "stocknonvat"); stockNonVAT != "" {
		if val, err := h.parseInt(stockNonVAT); err == nil {
			stock.NonVAT.Remaining = val
		}
	}

	// Parse actual stock
	if actualStock := h.getFieldValue(record, headerMap, "actualstock"); actualStock != "" {
		if val, err := h.parseInt(actualStock); err == nil {
			stock.ActualStock = val
		}
	} else {
		// If actual stock not provided, calculate from VAT + Non-VAT
		stock.ActualStock = stock.VAT.Remaining + stock.NonVAT.Remaining
	}

	return stock
}

// parseFloat parses string to float64
func (h *MigrationHandler) parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// parseInt parses string to int
func (h *MigrationHandler) parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(strings.TrimSpace(s))
}

// GetProductCSVTemplate returns a CSV template for product data
func (h *MigrationHandler) GetProductCSVTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create CSV template
	template := "skuId,name,description,color,size,category,purchasePriceVAT,purchasePriceNonVAT,salePriceVAT,salePriceNonVAT,stockVAT,stockNonVAT,actualStock\n"
	template += "SH-0001,เสื้อเชิ้ต,เสื้อเชิ้ตผ้าฝ้าย,ขาว,L,เสื้อผ้า,299.00,250.00,399.00,350.00,50,30,80\n"
	template += ",กางเกงยีนส์,กางเกงยีนส์สไตล์สตรีท,น้ำเงิน,32,กางเกง,599.00,500.00,799.00,650.00,25,15,40\n"
	template += "AC-0001,กระเป๋า,กระเป๋าหนังแท้,ดำ,One Size,กระเป๋า,1299.00,1100.00,1799.00,1500.00,10,5,15\n"

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=product_template.csv")
	w.Write([]byte(template))
}

// generateProductCode generates product code based on category, size, and color
func (h *MigrationHandler) generateProductCode(category, size, color string) string {
	// Get category prefix (first 2 characters)
	categoryPrefix := strings.ToUpper(category)
	if len(categoryPrefix) > 2 {
		categoryPrefix = categoryPrefix[:2]
	} else if len(categoryPrefix) == 1 {
		categoryPrefix = categoryPrefix + "X"
	} else if len(categoryPrefix) == 0 {
		categoryPrefix = "XX"
	}

	// Get size code (first 2 characters)
	sizeCode := strings.ToUpper(size)
	if len(sizeCode) > 2 {
		sizeCode = sizeCode[:2]
	} else if len(sizeCode) == 1 {
		sizeCode = sizeCode + "X"
	} else if len(sizeCode) == 0 {
		sizeCode = "XX"
	}

	// Get color code (first 2 characters)
	colorCode := strings.ToUpper(color)
	if len(colorCode) > 2 {
		colorCode = colorCode[:2]
	} else if len(colorCode) == 1 {
		colorCode = colorCode + "X"
	} else if len(colorCode) == 0 {
		colorCode = "XX"
	}

	// Format: Category-Size/Color
	return fmt.Sprintf("%s-%s/%s", categoryPrefix, sizeCode, colorCode)
}
