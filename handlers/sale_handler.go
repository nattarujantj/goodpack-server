package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/services"
	"goodpack-server/utils"
)

type SaleHandler struct {
	saleRepo            *repository.SaleRepository
	customerRepo        *repository.CustomerRepository
	productRepo         *repository.ProductRepository
	quotationRepo       *repository.QuotationRepository
	stockAdjustmentRepo *repository.StockAdjustmentRepository
	bankAccountService  *services.BankAccountService
}

func NewSaleHandler(saleRepo *repository.SaleRepository, customerRepo *repository.CustomerRepository, productRepo *repository.ProductRepository, quotationRepo *repository.QuotationRepository, stockAdjustmentRepo *repository.StockAdjustmentRepository) *SaleHandler {
	return &SaleHandler{
		saleRepo:            saleRepo,
		customerRepo:        customerRepo,
		productRepo:         productRepo,
		quotationRepo:       quotationRepo,
		stockAdjustmentRepo: stockAdjustmentRepo,
		bankAccountService:  services.NewBankAccountService(),
	}
}

// enrichSaleWithCustomerData enriches a sale with customer data
func (h *SaleHandler) enrichSaleWithCustomerData(sale *models.Sale) {
	customer, err := h.customerRepo.GetByID(sale.CustomerID)
	if err == nil {
		// Update sale with customer data
		sale.CustomerName = customer.CompanyName
		if sale.CustomerName == "" {
			sale.CustomerName = customer.ContactName
		}
		sale.ContactName = &customer.ContactName
		sale.CustomerCode = &customer.CustomerCode
		sale.TaxID = &customer.TaxID
		sale.Address = &customer.Address
		sale.Phone = &customer.Phone
	}
}

// enrichSaleWithBankAccountData enriches a sale with bank account data
func (h *SaleHandler) enrichSaleWithBankAccountData(sale *models.Sale) {
	if sale.Payment.OurAccount != nil && *sale.Payment.OurAccount != "" {
		bankAccount, err := h.bankAccountService.LoadBankAccountFromConfig(*sale.Payment.OurAccount)
		if err == nil && bankAccount != nil {
			sale.Payment.OurAccountInfo = bankAccount
		}
	}
}

// generateSaleID generates a unique sale ID based on VAT status.
// Sequence continues across months (e.g. INV-6901-0005 -> INV-6902-0006).
func (h *SaleHandler) generateSaleID(ctx context.Context, isVAT bool) (string, error) {
	now := utils.NowInThailand()
	beYear := now.Year() + 543
	dateStr := fmt.Sprintf("%02d%02d", beYear%100, int(now.Month())) // YYMM format

	// Base prefix for sequence lookup (no month - continue across months)
	var basePrefix string
	if isVAT {
		basePrefix = "INV-"
	} else {
		basePrefix = "NV-"
	}

	nextSeq, err := h.saleRepo.GetNextSequenceNumber(ctx, basePrefix)
	if err != nil {
		return "", err
	}

	seqStr := fmt.Sprintf("%04d", nextSeq)
	// Full prefix with current YYMM for display
	fullPrefix := fmt.Sprintf("INV-%s", dateStr)
	if !isVAT {
		fullPrefix = fmt.Sprintf("NV-%s", dateStr)
	}
	return fullPrefix + "-" + seqStr, nil
}

func (h *SaleHandler) GetSales(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	sales, err := h.saleRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch sales", http.StatusInternalServerError)
		return
	}

	// Enrich sales with customer data
	for i := range sales {
		h.enrichSaleWithCustomerData(&sales[i])
		h.enrichSaleWithBankAccountData(&sales[i])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sales)
}

func (h *SaleHandler) GetSale(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid sale ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	sale, err := h.saleRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Sale not found", http.StatusNotFound)
		return
	}

	// Enrich sale with customer data
	h.enrichSaleWithCustomerData(sale)
	h.enrichSaleWithBankAccountData(sale)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sale)
}

func (h *SaleHandler) CreateSale(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var saleReq models.SaleRequest
	if err := json.NewDecoder(r.Body).Decode(&saleReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var saleCode string
	if saleReq.SaleCode != nil && strings.TrimSpace(*saleReq.SaleCode) != "" {
		saleCode = strings.TrimSpace(*saleReq.SaleCode)
		existing, err := h.saleRepo.GetBySaleCode(ctx, saleCode)
		if err == nil && existing != nil {
			http.Error(w, "Sale code already exists", http.StatusBadRequest)
			return
		}
	} else {
		var err error
		saleCode, err = h.generateSaleID(ctx, saleReq.IsVAT)
		if err != nil {
			http.Error(w, "Failed to generate sale ID", http.StatusInternalServerError)
			return
		}
	}

	// Create sale
	sale := saleReq.ToSale()
	sale.SaleCode = saleCode

	// Cut stock for each item
	for _, item := range sale.Items {
		product, err := h.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Product not found: %s", item.ProductID), http.StatusBadRequest)
			return
		}

		// Update sale price using new UpdatePrice method with quantity
		product.UpdatePrice(item.UnitPrice, sale.IsVAT, false, item.Quantity) // false = isSale

		// Determine stock type based on VAT status
		var stockType models.StockType
		if sale.IsVAT {
			stockType = models.StockTypeVAT
		} else {
			stockType = models.StockTypeNonVAT
		}

		// Apply stock adjustment using centralized stock management logic
		ApplyStockAdjustment(product, models.AdjustmentTypeReduce, stockType, item.Quantity)

		// Update product
		if err := h.productRepo.Update(ctx, item.ProductID, product); err != nil {
			http.Error(w, fmt.Sprintf("Failed to update product stock: %s", item.ProductID), http.StatusInternalServerError)
			return
		}

		// Record stock change in history
		saleID := sale.ID.Hex()
		saleCode := sale.SaleCode
		notes := fmt.Sprintf("ขายจากรายการ %s", saleCode)
		if err := RecordStockChange(
			ctx,
			h.stockAdjustmentRepo,
			product,
			models.SourceTypeSale,
			&saleID,
			&saleCode,
			models.AdjustmentTypeReduce,
			stockType,
			item.Quantity,
			&notes,
		); err != nil {
			// Log error but don't fail the sale
			fmt.Printf("Warning: Failed to record stock change history: %v\n", err)
		}
	}

	// Save sale
	if err := h.saleRepo.Create(sale); err != nil {
		http.Error(w, "Failed to create sale", http.StatusInternalServerError)
		return
	}

	// Update quotation with sale code if quotationCode is provided
	if sale.QuotationCode != nil && *sale.QuotationCode != "" {
		if err := h.updateQuotationWithSaleCode(ctx, *sale.QuotationCode, sale.SaleCode); err != nil {
			// Log error but don't fail the sale creation
			fmt.Printf("Warning: Failed to update quotation %s with sale code %s: %v\n", *sale.QuotationCode, sale.SaleCode, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sale)
}

func (h *SaleHandler) UpdateSale(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid sale ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	var saleReq models.SaleRequest
	if err := json.NewDecoder(r.Body).Decode(&saleReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing sale
	existingSale, err := h.saleRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Sale not found", http.StatusNotFound)
		return
	}

	// If saleCode is being changed, validate uniqueness
	if saleReq.SaleCode != nil && strings.TrimSpace(*saleReq.SaleCode) != "" {
		newCode := strings.TrimSpace(*saleReq.SaleCode)
		if newCode != existingSale.SaleCode {
			other, err := h.saleRepo.GetBySaleCode(ctx, newCode)
			if err == nil && other != nil && other.ID.Hex() != id {
				http.Error(w, "Sale code already exists", http.StatusBadRequest)
				return
			}
		}
	}

	// Restore stock and rollback price updates for old items
	for _, item := range existingSale.Items {
		product, err := h.productRepo.GetByID(ctx, item.ProductID)
		if err == nil {
			var stockType models.StockType
			if existingSale.IsVAT {
				stockType = models.StockTypeVAT
			} else {
				stockType = models.StockTypeNonVAT
			}
			// Restore stock by adding back (reverse the reduce operation)
			ApplyStockAdjustment(product, models.AdjustmentTypeAdd, stockType, item.Quantity)
			// Rollback price update
			product.RollbackPriceUpdate(item.UnitPrice, existingSale.IsVAT, false, item.Quantity)
			h.productRepo.Update(ctx, item.ProductID, product)
		}
	}

	// Update sale
	existingSale.UpdateFromRequest(&saleReq)

	// Cut stock and update price for new items
	for _, item := range existingSale.Items {
		product, err := h.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Product not found: %s", item.ProductID), http.StatusBadRequest)
			return
		}

		// Update sale price using new UpdatePrice method with quantity
		product.UpdatePrice(item.UnitPrice, existingSale.IsVAT, false, item.Quantity) // false = isSale

		// Determine stock type based on VAT status
		var stockType models.StockType
		if existingSale.IsVAT {
			stockType = models.StockTypeVAT
		} else {
			stockType = models.StockTypeNonVAT
		}

		// Apply stock adjustment using centralized stock management logic
		ApplyStockAdjustment(product, models.AdjustmentTypeReduce, stockType, item.Quantity)

		// Update product
		if err := h.productRepo.Update(ctx, item.ProductID, product); err != nil {
			http.Error(w, fmt.Sprintf("Failed to update product stock: %s", item.ProductID), http.StatusInternalServerError)
			return
		}
	}

	// Save updated sale
	if err := h.saleRepo.Update(id, existingSale); err != nil {
		http.Error(w, "Failed to update sale", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingSale)
}

func (h *SaleHandler) DeleteSale(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid sale ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	// Get existing sale to restore stock
	existingSale, err := h.saleRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Sale not found", http.StatusNotFound)
		return
	}

	// Restore stock and rollback price updates for all items
	for _, item := range existingSale.Items {
		product, err := h.productRepo.GetByID(ctx, item.ProductID)
		if err == nil {
			var stockType models.StockType
			if existingSale.IsVAT {
				stockType = models.StockTypeVAT
			} else {
				stockType = models.StockTypeNonVAT
			}
			// Restore stock by adding back (reverse the reduce operation)
			ApplyStockAdjustment(product, models.AdjustmentTypeAdd, stockType, item.Quantity)
			// Rollback price update
			product.RollbackPriceUpdate(item.UnitPrice, existingSale.IsVAT, false, item.Quantity)
			h.productRepo.Update(ctx, item.ProductID, product)
		}
	}

	// Delete sale
	if err := h.saleRepo.Delete(id); err != nil {
		http.Error(w, "Failed to delete sale", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// updateQuotationWithSaleCode updates a quotation with the sale code
func (h *SaleHandler) updateQuotationWithSaleCode(ctx context.Context, quotationCode, saleCode string) error {
	// Find quotation by code
	quotation, err := h.quotationRepo.GetByCode(quotationCode)
	if err != nil {
		return fmt.Errorf("quotation not found: %w", err)
	}

	// Update quotation with sale code
	quotation.SaleCode = &saleCode
	quotation.UpdatedAt = utils.NowInThailand()

	// Save updated quotation
	return h.quotationRepo.Update(quotation.ID.Hex(), quotation)
}
