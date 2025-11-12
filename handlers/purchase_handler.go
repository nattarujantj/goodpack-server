package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type PurchaseHandler struct {
	purchaseRepo        *repository.PurchaseRepository
	customerRepo        *repository.CustomerRepository
	productRepo         *repository.ProductRepository
	stockAdjustmentRepo *repository.StockAdjustmentRepository
}

func NewPurchaseHandler(purchaseRepo *repository.PurchaseRepository, customerRepo *repository.CustomerRepository, productRepo *repository.ProductRepository, stockAdjustmentRepo *repository.StockAdjustmentRepository) *PurchaseHandler {
	return &PurchaseHandler{
		purchaseRepo:        purchaseRepo,
		customerRepo:        customerRepo,
		productRepo:         productRepo,
		stockAdjustmentRepo: stockAdjustmentRepo,
	}
}

// enrichPurchaseWithCustomerData enriches a purchase with customer data
func (h *PurchaseHandler) enrichPurchaseWithCustomerData(purchase *models.Purchase) {
	customer, err := h.customerRepo.GetByID(purchase.CustomerID)
	if err == nil {
		// Update purchase with customer data
		purchase.CustomerName = customer.CompanyName
		if purchase.CustomerName == "" {
			purchase.CustomerName = customer.ContactName
		}
		purchase.ContactName = &customer.ContactName
		purchase.CustomerCode = &customer.CustomerCode
		purchase.TaxID = &customer.TaxID
		purchase.Address = &customer.Address
		purchase.Phone = &customer.Phone
	}
}

// generatePurchaseID generates a unique purchase ID based on VAT status
func (h *PurchaseHandler) generatePurchaseID(ctx context.Context, isVAT bool) (string, error) {
	now := time.Now()
	// Convert to Buddhist Era (BE)
	beYear := now.Year() + 543
	dateStr := fmt.Sprintf("%02d%02d", beYear%100, int(now.Month())) // YYMM format

	var prefix string
	if isVAT {
		prefix = fmt.Sprintf("PUR-VAT-%s", dateStr)
	} else {
		prefix = fmt.Sprintf("PUR-NV-%s", dateStr)
	}

	// Get the next sequence number for this prefix
	nextSeq, err := h.purchaseRepo.GetNextSequenceNumber(ctx, prefix)
	if err != nil {
		return "", err
	}

	// Format sequence number with leading zeros (4 digits)
	seqStr := fmt.Sprintf("%04d", nextSeq)

	return prefix + "-" + seqStr, nil
}

func (h *PurchaseHandler) GetPurchases(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	purchases, err := h.purchaseRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch purchases", http.StatusInternalServerError)
		return
	}

	// Enrich purchases with customer data
	for i := range purchases {
		h.enrichPurchaseWithCustomerData(purchases[i])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(purchases)
}

func (h *PurchaseHandler) GetPurchase(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid purchase ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	purchase, err := h.purchaseRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Purchase not found", http.StatusNotFound)
		return
	}

	// Enrich purchase with customer data
	h.enrichPurchaseWithCustomerData(purchase)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(purchase)
}

func (h *PurchaseHandler) CreatePurchase(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Read request body for debugging
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Log request body
	fmt.Printf("Request body: %s\n", string(body))

	// Create new reader from body
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	var purchaseRequest models.PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&purchaseRequest); err != nil {
		fmt.Printf("JSON decode error: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get customer name
	customer, err := h.customerRepo.GetByID(purchaseRequest.CustomerID)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusBadRequest)
		return
	}

	purchase := purchaseRequest.ToPurchase()
	purchase.CustomerName = customer.CompanyName
	if purchase.CustomerName == "" {
		purchase.CustomerName = customer.ContactName
	}
	purchase.ContactName = &customer.ContactName

	// Generate unique purchase code
	purchaseCode, err := h.generatePurchaseID(ctx, purchase.IsVAT)
	if err != nil {
		http.Error(w, "Failed to generate purchase code", http.StatusInternalServerError)
		return
	}
	purchase.PurchaseCode = purchaseCode

	// Create purchase
	if err := h.purchaseRepo.Create(ctx, purchase); err != nil {
		http.Error(w, "Failed to create purchase", http.StatusInternalServerError)
		return
	}

	// Update product prices and stock
	if err := h.updateProductData(ctx, purchase); err != nil {
		// Log error but don't fail the purchase creation
		// TODO: Add proper logging
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(purchase)
}

func (h *PurchaseHandler) UpdatePurchase(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid purchase ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	// Get existing purchase
	existingPurchase, err := h.purchaseRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Purchase not found", http.StatusNotFound)
		return
	}

	var purchaseRequest models.PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&purchaseRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get customer name
	customer, err := h.customerRepo.GetByID(purchaseRequest.CustomerID)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusBadRequest)
		return
	}

	// Update purchase
	existingPurchase.UpdateFromRequest(&purchaseRequest)
	existingPurchase.CustomerName = customer.CompanyName
	if existingPurchase.CustomerName == "" {
		existingPurchase.CustomerName = customer.ContactName
	}

	if err := h.purchaseRepo.Update(ctx, id, existingPurchase); err != nil {
		http.Error(w, "Failed to update purchase", http.StatusInternalServerError)
		return
	}

	// Update product prices and stock
	if err := h.updateProductData(ctx, existingPurchase); err != nil {
		// Log error but don't fail the purchase update
		// TODO: Add proper logging
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingPurchase)
}

func (h *PurchaseHandler) DeletePurchase(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid purchase ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	if err := h.purchaseRepo.Delete(ctx, id); err != nil {
		http.Error(w, "Failed to delete purchase", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PurchaseHandler) updateProductData(ctx context.Context, purchase *models.Purchase) error {
	// Update product prices and stock for each item
	for _, item := range purchase.Items {
		product, err := h.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			continue // Skip if product not found
		}

		// Update purchase price using new UpdatePrice method
		product.UpdatePrice(item.UnitPrice, purchase.IsVAT, true) // true = isPurchase

		// Determine stock type based on VAT status
		var stockType models.StockType
		if purchase.IsVAT {
			stockType = models.StockTypeVAT
		} else {
			stockType = models.StockTypeNonVAT
		}

		// Apply stock adjustment using centralized stock management logic
		ApplyStockAdjustment(product, models.AdjustmentTypeAdd, stockType, item.Quantity)

		// Save updated product
		if err := h.productRepo.Update(ctx, item.ProductID, product); err != nil {
			continue
		}

		// Record stock change in history
		purchaseID := purchase.ID.Hex()
		purchaseCode := purchase.PurchaseCode
		notes := fmt.Sprintf("ซื้อจากรายการ %s", purchaseCode)
		if err := RecordStockChange(
			ctx,
			h.stockAdjustmentRepo,
			product,
			models.SourceTypePurchase,
			&purchaseID,
			&purchaseCode,
			models.AdjustmentTypeAdd,
			stockType,
			item.Quantity,
			&notes,
		); err != nil {
			// Log error but don't fail the purchase
			fmt.Printf("Warning: Failed to record stock change history: %v\n", err)
		}
	}

	return nil
}
