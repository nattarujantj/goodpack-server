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
	purchaseRepo *repository.PurchaseRepository
	saleRepo     *repository.SaleRepository
}

func NewMigrationHandler(customerRepo *repository.CustomerRepository, productRepo *repository.ProductRepository, purchaseRepo *repository.PurchaseRepository, saleRepo *repository.SaleRepository) *MigrationHandler {
	return &MigrationHandler{
		customerRepo: customerRepo,
		productRepo:  productRepo,
		purchaseRepo: purchaseRepo,
		saleRepo:     saleRepo,
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

// PurchaseCSVRow represents a row in the purchase CSV file
type PurchaseCSVRow struct {
	PurchaseCode string `csv:"purchaseCode"`
	PurchaseDate string `csv:"purchaseDate"`
	CustomerCode string `csv:"customerCode"`
	ProductCode  string `csv:"productCode"`
	Quantity     string `csv:"quantity"`
	UnitPrice    string `csv:"unitPrice"`
	IsVAT        string `csv:"isVAT"`
	ShippingCost string `csv:"shippingCost"`
	Notes        string `csv:"notes"`
}

// SaleCSVRow represents a row in the sale CSV file
type SaleCSVRow struct {
	SaleCode     string `csv:"saleCode"`
	SaleDate     string `csv:"saleDate"`
	CustomerCode string `csv:"customerCode"`
	ProductCode  string `csv:"productCode"`
	Quantity     string `csv:"quantity"`
	UnitPrice    string `csv:"unitPrice"`
	IsVAT        string `csv:"isVAT"`
	ShippingCost string `csv:"shippingCost"`
	Notes        string `csv:"notes"`
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

// MigratePurchasesFromCSV handles CSV file upload and migration for purchases
func (h *MigrationHandler) MigratePurchasesFromCSV(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.parseAndMigratePurchaseCSV(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process CSV: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// parseAndMigratePurchaseCSV parses CSV file and migrates purchase data to database
func (h *MigrationHandler) parseAndMigratePurchaseCSV(file io.Reader) (*MigrationResult, error) {
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
	requiredHeaders := []string{"purchasedate", "customercode", "productcode", "quantity", "unitprice"}
	optionalHeaders := []string{"purchasecode", "isvat", "shippingcost", "notes"}

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

	// Group records by purchase (same purchaseCode or purchaseDate + customerCode)
	purchaseGroups := h.groupPurchaseRecords(records[1:], headerMap)

	// Process each purchase group
	for groupKey, groupRecords := range purchaseGroups {
		rowNum := groupRecords[0].RowNum

		// Create purchase from CSV group
		purchase, err := h.createPurchaseFromGroup(groupRecords, headerMap)
		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Group %s: %v", groupKey, err))
			continue
		}

		// Validate required fields
		if purchase.CustomerID == "" {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Customer not found", rowNum))
			continue
		}

		if len(purchase.Items) == 0 {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: No valid items found", rowNum))
			continue
		}

		// Save to database
		err = h.purchaseRepo.Create(context.Background(), purchase)
		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to save purchase - %v", rowNum, err))
			continue
		}

		// Update product prices and stock
		err = h.updateProductsFromPurchase(purchase)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to update products - %v", rowNum, err))
		}

		result.SuccessRows++
	}

	return result, nil
}

// PurchaseRecord represents a single CSV record with row number
type PurchaseRecord struct {
	RowNum int
	Record []string
}

// groupPurchaseRecords groups CSV records by purchase
func (h *MigrationHandler) groupPurchaseRecords(records [][]string, headerMap map[string]int) map[string][]PurchaseRecord {
	groups := make(map[string][]PurchaseRecord)

	for i, record := range records {
		rowNum := i + 2 // +2 because we start from row 2 (after header)

		// Get group key (purchaseCode or purchaseDate + customerCode)
		purchaseCode := h.getFieldValue(record, headerMap, "purchasecode")
		purchaseDate := h.getFieldValue(record, headerMap, "purchasedate")
		customerCode := h.getFieldValue(record, headerMap, "customercode")

		var groupKey string
		if purchaseCode != "" {
			groupKey = purchaseCode
		} else {
			groupKey = fmt.Sprintf("%s-%s", purchaseDate, customerCode)
		}

		groups[groupKey] = append(groups[groupKey], PurchaseRecord{
			RowNum: rowNum,
			Record: record,
		})
	}

	return groups
}

// createPurchaseFromGroup creates a purchase from a group of CSV records
func (h *MigrationHandler) createPurchaseFromGroup(records []PurchaseRecord, headerMap map[string]int) (*models.Purchase, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records in group")
	}

	firstRecord := records[0].Record

	// Parse purchase date
	purchaseDateStr := h.getFieldValue(firstRecord, headerMap, "purchasedate")
	purchaseDate, err := time.Parse("2006-01-02", purchaseDateStr)
	if err != nil {
		purchaseDate = time.Now() // Default to current time if parsing fails
	}

	// Get customer by code
	customerCode := h.getFieldValue(firstRecord, headerMap, "customercode")
	customer, err := h.customerRepo.GetByCustomerCode(customerCode)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %s", customerCode)
	}

	// Parse VAT status
	isVAT := strings.ToLower(h.getFieldValue(firstRecord, headerMap, "isvat")) == "true"

	// Parse shipping cost
	shippingCost, _ := h.parseFloat(h.getFieldValue(firstRecord, headerMap, "shippingcost"))

	// Parse notes
	notes := h.getFieldValue(firstRecord, headerMap, "notes")
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	// Create purchase items
	var items []models.PurchaseItem
	for _, record := range records {
		productCode := h.getFieldValue(record.Record, headerMap, "productcode")
		quantityStr := h.getFieldValue(record.Record, headerMap, "quantity")
		unitPriceStr := h.getFieldValue(record.Record, headerMap, "unitprice")

		// Get product by code
		product, err := h.productRepo.GetByCode(context.Background(), productCode)
		if err != nil {
			return nil, fmt.Errorf("product not found: %s", productCode)
		}

		quantity, err := h.parseInt(quantityStr)
		if err != nil {
			return nil, fmt.Errorf("invalid quantity: %s", quantityStr)
		}

		unitPrice, err := h.parseFloat(unitPriceStr)
		if err != nil {
			return nil, fmt.Errorf("invalid unit price: %s", unitPriceStr)
		}

		totalPrice := unitPrice * float64(quantity)

		items = append(items, models.PurchaseItem{
			ProductID:   product.ID.Hex(),
			ProductName: product.Name,
			ProductCode: product.Code,
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  totalPrice,
		})
	}

	// Calculate totals
	var totalAmount float64
	for _, item := range items {
		totalAmount += item.TotalPrice
	}

	var totalVAT float64
	if isVAT {
		totalVAT = totalAmount * 0.07 // 7% VAT
	}

	grandTotal := totalAmount + totalVAT + shippingCost

	// Generate purchase code if not provided
	purchaseCode := h.getFieldValue(firstRecord, headerMap, "purchasecode")
	if purchaseCode == "" {
		purchaseCode, err = h.generatePurchaseCode(isVAT)
		if err != nil {
			return nil, fmt.Errorf("failed to generate purchase code: %v", err)
		}
	}

	// Create purchase
	purchase := &models.Purchase{
		PurchaseCode: purchaseCode,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		PurchaseDate: purchaseDate,
		CustomerID:   customer.ID.Hex(),
		CustomerName: customer.CompanyName,
		ContactName:  &customer.ContactName,
		CustomerCode: &customer.CustomerCode,
		TaxID:        &customer.TaxID,
		Address:      &customer.Address,
		Phone:        &customer.Phone,
		Notes:        notesPtr,
		Items:        items,
		IsVAT:        isVAT,
		ShippingCost: shippingCost,
		Payment: models.PaymentInfo{
			IsPaid: false,
		},
		Warehouse: models.WarehouseInfo{
			IsUpdated:      false,
			ActualShipping: shippingCost,
		},
		TotalAmount: totalAmount,
		TotalVAT:    totalVAT,
		GrandTotal:  grandTotal,
	}

	return purchase, nil
}

// updateProductsFromPurchase updates product prices and stock based on purchase
func (h *MigrationHandler) updateProductsFromPurchase(purchase *models.Purchase) error {
	for _, item := range purchase.Items {
		// Get product
		product, err := h.productRepo.GetByID(context.Background(), item.ProductID)
		if err != nil {
			return fmt.Errorf("failed to get product %s: %v", item.ProductID, err)
		}

		// Update price
		product.UpdatePrice(item.UnitPrice, purchase.IsVAT, true) // true = isPurchase

		// Update stock
		if purchase.IsVAT {
			product.Stock.VAT.Purchased += item.Quantity
			product.Stock.VAT.Remaining += item.Quantity
		} else {
			product.Stock.NonVAT.Purchased += item.Quantity
			product.Stock.NonVAT.Remaining += item.Quantity
		}
		product.Stock.ActualStock += item.Quantity

		// Save updated product
		err = h.productRepo.Update(context.Background(), item.ProductID, product)
		if err != nil {
			return fmt.Errorf("failed to update product %s: %v", item.ProductID, err)
		}
	}

	return nil
}

// generatePurchaseCode generates a unique purchase code
func (h *MigrationHandler) generatePurchaseCode(isVAT bool) (string, error) {
	// This is a simplified version - you might want to use the actual repository method
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()

	prefix := "P"
	if isVAT {
		prefix = "PV"
	}

	// Simple format: P-YYYYMMDD-001 or PV-YYYYMMDD-001
	return fmt.Sprintf("%s-%04d%02d%02d-001", prefix, year, month, day), nil
}

// GetPurchaseCSVTemplate returns a CSV template for purchase data
func (h *MigrationHandler) GetPurchaseCSVTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create CSV template
	template := "purchaseCode,purchaseDate,customerCode,productCode,quantity,unitPrice,isVAT,shippingCost,notes\n"
	template += "P-001,2024-01-15,C-0001,เ-l/WH,10,299.00,true,50.00,ซื้อเสื้อเชิ้ต\n"
	template += ",2024-01-15,C-0001,ก-32/BL,5,599.00,true,,ซื้อกางเกงยีนส์\n"
	template += "P-002,2024-01-16,C-0002,ก-onesize/BK,2,1299.00,false,100.00,ซื้อกระเป๋า\n"

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=purchase_template.csv")
	w.Write([]byte(template))
}

// MigrateSalesFromCSV handles CSV file upload and migration for sales
func (h *MigrationHandler) MigrateSalesFromCSV(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.parseAndMigrateSaleCSV(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process CSV: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// parseAndMigrateSaleCSV parses CSV file and migrates sale data to database
func (h *MigrationHandler) parseAndMigrateSaleCSV(file io.Reader) (*MigrationResult, error) {
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
	requiredHeaders := []string{"saledate", "customercode", "productcode", "quantity", "unitprice"}
	optionalHeaders := []string{"salecode", "isvat", "shippingcost", "notes"}

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

	// Group records by sale (same saleCode or saleDate + customerCode)
	saleGroups := h.groupSaleRecords(records[1:], headerMap)

	// Process each sale group
	for groupKey, groupRecords := range saleGroups {
		rowNum := groupRecords[0].RowNum

		// Create sale from CSV group
		sale, err := h.createSaleFromGroup(groupRecords, headerMap)
		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Group %s: %v", groupKey, err))
			continue
		}

		// Validate required fields
		if sale.CustomerID == "" {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Customer not found", rowNum))
			continue
		}

		if len(sale.Items) == 0 {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: No valid items found", rowNum))
			continue
		}

		// Save to database
		err = h.saleRepo.Create(sale)
		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to save sale - %v", rowNum, err))
			continue
		}

		// Update product prices and stock
		err = h.updateProductsFromSale(sale)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Failed to update products - %v", rowNum, err))
		}

		result.SuccessRows++
	}

	return result, nil
}

// SaleRecord represents a single CSV record with row number
type SaleRecord struct {
	RowNum int
	Record []string
}

// groupSaleRecords groups CSV records by sale
func (h *MigrationHandler) groupSaleRecords(records [][]string, headerMap map[string]int) map[string][]SaleRecord {
	groups := make(map[string][]SaleRecord)

	for i, record := range records {
		rowNum := i + 2 // +2 because we start from row 2 (after header)

		// Get group key (saleCode or saleDate + customerCode)
		saleCode := h.getFieldValue(record, headerMap, "salecode")
		saleDate := h.getFieldValue(record, headerMap, "saledate")
		customerCode := h.getFieldValue(record, headerMap, "customercode")

		var groupKey string
		if saleCode != "" {
			groupKey = saleCode
		} else {
			groupKey = fmt.Sprintf("%s-%s", saleDate, customerCode)
		}

		groups[groupKey] = append(groups[groupKey], SaleRecord{
			RowNum: rowNum,
			Record: record,
		})
	}

	return groups
}

// createSaleFromGroup creates a sale from a group of CSV records
func (h *MigrationHandler) createSaleFromGroup(records []SaleRecord, headerMap map[string]int) (*models.Sale, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records in group")
	}

	firstRecord := records[0].Record

	// Parse sale date
	saleDateStr := h.getFieldValue(firstRecord, headerMap, "saledate")
	saleDate, err := time.Parse("2006-01-02", saleDateStr)
	if err != nil {
		saleDate = time.Now() // Default to current time if parsing fails
	}

	// Get customer by code
	customerCode := h.getFieldValue(firstRecord, headerMap, "customercode")
	customer, err := h.customerRepo.GetByCustomerCode(customerCode)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %s", customerCode)
	}

	// Parse VAT status
	isVAT := strings.ToLower(h.getFieldValue(firstRecord, headerMap, "isvat")) == "true"

	// Parse shipping cost
	shippingCost, _ := h.parseFloat(h.getFieldValue(firstRecord, headerMap, "shippingcost"))

	// Parse notes
	notes := h.getFieldValue(firstRecord, headerMap, "notes")
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	// Create sale items
	var items []models.SaleItem
	for _, record := range records {
		productCode := h.getFieldValue(record.Record, headerMap, "productcode")
		quantityStr := h.getFieldValue(record.Record, headerMap, "quantity")
		unitPriceStr := h.getFieldValue(record.Record, headerMap, "unitprice")

		// Get product by code
		product, err := h.productRepo.GetByCode(context.Background(), productCode)
		if err != nil {
			return nil, fmt.Errorf("product not found: %s", productCode)
		}

		quantity, err := h.parseInt(quantityStr)
		if err != nil {
			return nil, fmt.Errorf("invalid quantity: %s", quantityStr)
		}

		unitPrice, err := h.parseFloat(unitPriceStr)
		if err != nil {
			return nil, fmt.Errorf("invalid unit price: %s", unitPriceStr)
		}

		totalPrice := unitPrice * float64(quantity)

		items = append(items, models.SaleItem{
			ProductID:   product.ID.Hex(),
			ProductName: product.Name,
			ProductCode: product.Code,
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  totalPrice,
		})
	}

	// Generate sale code if not provided
	saleCode := h.getFieldValue(firstRecord, headerMap, "salecode")
	if saleCode == "" {
		saleCode, err = h.generateSaleCode(isVAT)
		if err != nil {
			return nil, fmt.Errorf("failed to generate sale code: %v", err)
		}
	}

	// Create sale
	sale := &models.Sale{
		SaleCode:     saleCode,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		SaleDate:     saleDate,
		CustomerID:   customer.ID.Hex(),
		CustomerName: customer.CompanyName,
		ContactName:  &customer.ContactName,
		CustomerCode: &customer.CustomerCode,
		TaxID:        &customer.TaxID,
		Address:      &customer.Address,
		Phone:        &customer.Phone,
		Items:        items,
		IsVAT:        isVAT,
		ShippingCost: shippingCost,
		Payment: models.PaymentInfo{
			IsPaid: false,
		},
		Warehouse: models.WarehouseInfo{
			IsUpdated:      false,
			ActualShipping: shippingCost,
		},
		Notes: notesPtr,
	}

	return sale, nil
}

// updateProductsFromSale updates product prices and stock based on sale
func (h *MigrationHandler) updateProductsFromSale(sale *models.Sale) error {
	for _, item := range sale.Items {
		// Get product
		product, err := h.productRepo.GetByID(context.Background(), item.ProductID)
		if err != nil {
			return fmt.Errorf("failed to get product %s: %v", item.ProductID, err)
		}

		// Update price
		product.UpdatePrice(item.UnitPrice, sale.IsVAT, false) // false = isSale

		// Update stock - reduce remaining stock
		if sale.IsVAT {
			product.Stock.VAT.Sold += item.Quantity
			product.Stock.VAT.Remaining -= item.Quantity
		} else {
			product.Stock.NonVAT.Sold += item.Quantity
			product.Stock.NonVAT.Remaining -= item.Quantity
		}
		product.Stock.ActualStock -= item.Quantity

		// Ensure stock doesn't go negative
		if product.Stock.ActualStock < 0 {
			product.Stock.ActualStock = 0
		}
		if product.Stock.VAT.Remaining < 0 {
			product.Stock.VAT.Remaining = 0
		}
		if product.Stock.NonVAT.Remaining < 0 {
			product.Stock.NonVAT.Remaining = 0
		}

		// Save updated product
		err = h.productRepo.Update(context.Background(), item.ProductID, product)
		if err != nil {
			return fmt.Errorf("failed to update product %s: %v", item.ProductID, err)
		}
	}

	return nil
}

// generateSaleCode generates a unique sale code
func (h *MigrationHandler) generateSaleCode(isVAT bool) (string, error) {
	// This is a simplified version - you might want to use the actual repository method
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()

	prefix := "S"
	if isVAT {
		prefix = "SV"
	}

	// Simple format: S-YYYYMMDD-001 or SV-YYYYMMDD-001
	return fmt.Sprintf("%s-%04d%02d%02d-001", prefix, year, month, day), nil
}

// GetSaleCSVTemplate returns a CSV template for sale data
func (h *MigrationHandler) GetSaleCSVTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create CSV template
	template := "saleCode,saleDate,customerCode,productCode,quantity,unitPrice,isVAT,shippingCost,notes\n"
	template += "S-001,2024-01-20,C-0001,เ-l/WH,5,399.00,true,30.00,ขายเสื้อเชิ้ต\n"
	template += ",2024-01-20,C-0001,ก-32/BL,2,799.00,true,,ขายกางเกงยีนส์\n"
	template += "S-002,2024-01-21,C-0002,ก-onesize/BK,1,1799.00,false,50.00,ขายกระเป๋า\n"

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=sale_template.csv")
	w.Write([]byte(template))
}
