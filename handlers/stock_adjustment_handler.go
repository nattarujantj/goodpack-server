package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type StockAdjustmentHandler struct {
	adjustmentRepo *repository.StockAdjustmentRepository
	productRepo    *repository.ProductRepository
}

func NewStockAdjustmentHandler(adjustmentRepo *repository.StockAdjustmentRepository, productRepo *repository.ProductRepository) *StockAdjustmentHandler {
	return &StockAdjustmentHandler{
		adjustmentRepo: adjustmentRepo,
		productRepo:    productRepo,
	}
}

// ApplyStockAdjustment applies stock adjustment to a product (core logic)
// Allows negative stock values to indicate abnormal stock status
func ApplyStockAdjustment(
	product *models.Product,
	adjustmentType models.StockAdjustmentType,
	stockType models.StockType,
	quantity int,
) {
	if stockType == models.StockTypeActualStock {
		// For ActualStock, just add or subtract directly
		// Allow negative values to show abnormal stock status
		if adjustmentType == models.AdjustmentTypeAdd {
			product.Stock.ActualStock += quantity
		} else {
			product.Stock.ActualStock -= quantity
			// Allow negative stock to indicate abnormal status
		}
	} else {
		// For VAT or NonVAT
		var stockInfo *models.StockInfo
		if stockType == models.StockTypeVAT {
			stockInfo = &product.Stock.VAT
		} else {
			stockInfo = &product.Stock.NonVAT
		}

		if adjustmentType == models.AdjustmentTypeAdd {
			// เพิ่ม: บวกใน Purchased และ Remaining
			stockInfo.Purchased += quantity
			stockInfo.Remaining += quantity
			// Also update ActualStock
			product.Stock.ActualStock += quantity
		} else {
			// ลด: บวกใน Sold และลด Remaining
			stockInfo.Sold += quantity
			stockInfo.Remaining -= quantity
			// Allow negative remaining to indicate abnormal status
			// Also update ActualStock
			product.Stock.ActualStock -= quantity
			// Allow negative ActualStock to indicate abnormal status
		}
	}
}

// RecordStockChange records a stock change in history (helper function for other handlers)
func RecordStockChange(
	ctx context.Context,
	adjustmentRepo *repository.StockAdjustmentRepository,
	product *models.Product,
	sourceType models.SourceType,
	sourceID, sourceCode *string,
	adjustmentType models.StockAdjustmentType,
	stockType models.StockType,
	quantity int,
	notes *string,
) error {
	adjustment := &models.StockAdjustment{
		ProductID:      product.ID.Hex(),
		ProductName:    product.Name,
		SKUID:          product.SKUID,
		AdjustmentType: adjustmentType,
		StockType:      stockType,
		Quantity:       quantity,
		SourceType:     sourceType,
		SourceID:       sourceID,
		SourceCode:     sourceCode,
		Notes:          notes,
		CreatedAt:      time.Now(),
	}

	// Store before values
	adjustment.BeforeVATPurchased = product.Stock.VAT.Purchased
	adjustment.BeforeVATSold = product.Stock.VAT.Sold
	adjustment.BeforeVATRemaining = product.Stock.VAT.Remaining
	adjustment.BeforeNonVATPurchased = product.Stock.NonVAT.Purchased
	adjustment.BeforeNonVATSold = product.Stock.NonVAT.Sold
	adjustment.BeforeNonVATRemaining = product.Stock.NonVAT.Remaining
	adjustment.BeforeActualStock = product.Stock.ActualStock

	// Set after values
	adjustment.SetAfterValues(product)

	// Save adjustment history
	return adjustmentRepo.Create(ctx, adjustment)
}

// AdjustStock handles stock adjustment request
func (h *StockAdjustmentHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	productID := vars["id"]

	// Get product - try by ObjectID first, then by SKUID
	product, err := h.productRepo.GetByID(ctx, productID)
	if err != nil {
		product, err = h.productRepo.GetBySKUID(ctx, productID)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	// Parse request
	var req models.StockAdjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Quantity <= 0 {
		http.Error(w, "Quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	if req.AdjustmentType != models.AdjustmentTypeAdd && req.AdjustmentType != models.AdjustmentTypeReduce {
		http.Error(w, "Invalid adjustment type. Must be 'add' or 'reduce'", http.StatusBadRequest)
		return
	}

	if req.StockType != models.StockTypeVAT && req.StockType != models.StockTypeNonVAT && req.StockType != models.StockTypeActualStock {
		http.Error(w, "Invalid stock type. Must be 'vat', 'nonvat', or 'actualstock'", http.StatusBadRequest)
		return
	}

	// Create adjustment record (before values)
	adjustment := req.ToStockAdjustment(product, models.SourceTypeAdjustment, nil, nil)

	// Apply stock adjustment using centralized logic
	ApplyStockAdjustment(product, req.AdjustmentType, req.StockType, req.Quantity)

	// Update product
	product.UpdatedAt = time.Now()
	if err := h.productRepo.Update(ctx, product.ID.Hex(), product); err != nil {
		http.Error(w, "Failed to update product stock", http.StatusInternalServerError)
		return
	}

	// Set after values in adjustment record
	adjustment.SetAfterValues(product)

	// Save adjustment history
	if err := h.adjustmentRepo.Create(ctx, adjustment); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to save stock adjustment history: %v\n", err)
	}

	// Return updated product
	json.NewEncoder(w).Encode(product)
}

// GetStockHistory gets stock adjustment history for a product
func (h *StockAdjustmentHandler) GetStockHistory(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	productID := vars["id"]

	// Get product to verify it exists
	_, err := h.productRepo.GetByID(ctx, productID)
	if err != nil {
		_, err = h.productRepo.GetBySKUID(ctx, productID)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	// Get limit from query parameter (default: 50)
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get date range from query parameters (optional)
	var startDate, endDate time.Time
	if startDateStr := r.URL.Query().Get("startDate"); startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}
	if endDateStr := r.URL.Query().Get("endDate"); endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Set to end of day
			endDate = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}

	var adjustments []*models.StockAdjustment
	if !startDate.IsZero() || !endDate.IsZero() {
		// Use date range if provided
		if startDate.IsZero() {
			startDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		if endDate.IsZero() {
			endDate = time.Now()
		}
		adjustments, err = h.adjustmentRepo.GetByProductIDAndDateRange(ctx, productID, startDate, endDate, limit)
	} else {
		// Get all adjustments for product
		adjustments, err = h.adjustmentRepo.GetByProductID(ctx, productID, limit)
	}

	if err != nil {
		http.Error(w, "Failed to get stock history", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(adjustments)
}

// GetAllStockHistory gets all stock adjustments across all products
func (h *StockAdjustmentHandler) GetAllStockHistory(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")

	// Get limit and skip from query parameters
	limit := 50
	skip := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if skipStr := r.URL.Query().Get("skip"); skipStr != "" {
		if parsedSkip, err := strconv.Atoi(skipStr); err == nil && parsedSkip >= 0 {
			skip = parsedSkip
		}
	}

	adjustments, err := h.adjustmentRepo.GetAll(ctx, limit, skip)
	if err != nil {
		http.Error(w, "Failed to get stock history", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(adjustments)
}

// GetStockHistoryBySource gets stock adjustments by source type and source ID
func (h *StockAdjustmentHandler) GetStockHistoryBySource(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")

	// Get source type and source ID from query parameters
	sourceTypeStr := r.URL.Query().Get("sourceType")
	sourceID := r.URL.Query().Get("sourceId")

	if sourceTypeStr == "" || sourceID == "" {
		http.Error(w, "sourceType and sourceId are required", http.StatusBadRequest)
		return
	}

	sourceType := models.SourceType(sourceTypeStr)
	if sourceType != models.SourceTypePurchase && sourceType != models.SourceTypeSale &&
		sourceType != models.SourceTypeAdjustment && sourceType != models.SourceTypeMigration {
		http.Error(w, "Invalid source type", http.StatusBadRequest)
		return
	}

	adjustments, err := h.adjustmentRepo.GetBySource(ctx, sourceType, sourceID)
	if err != nil {
		http.Error(w, "Failed to get stock history", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(adjustments)
}

// DeleteStockAdjustment deletes a stock adjustment and reverses the stock change
func (h *StockAdjustmentHandler) DeleteStockAdjustment(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	adjustmentID := vars["id"]

	// Get the adjustment record
	adjustment, err := h.adjustmentRepo.GetByID(ctx, adjustmentID)
	if err != nil {
		http.Error(w, "Stock adjustment not found", http.StatusNotFound)
		return
	}

	// Get the product
	product, err := h.productRepo.GetByID(ctx, adjustment.ProductID)
	if err != nil {
		http.Error(w, "Product not found", http.StatusInternalServerError)
		return
	}

	// Reverse the stock adjustment
	// If it was "add", we need to "reduce"
	// If it was "reduce", we need to "add"
	var reverseType models.StockAdjustmentType
	if adjustment.AdjustmentType == models.AdjustmentTypeAdd {
		reverseType = models.AdjustmentTypeReduce
	} else {
		reverseType = models.AdjustmentTypeAdd
	}

	// Apply reverse adjustment
	ApplyStockAdjustment(product, reverseType, adjustment.StockType, adjustment.Quantity)

	// Update product
	product.UpdatedAt = time.Now()
	if err := h.productRepo.Update(ctx, product.ID.Hex(), product); err != nil {
		http.Error(w, "Failed to update product stock", http.StatusInternalServerError)
		return
	}

	// Delete the adjustment record
	if err := h.adjustmentRepo.Delete(ctx, adjustmentID); err != nil {
		http.Error(w, "Failed to delete stock adjustment", http.StatusInternalServerError)
		return
	}

	// Return updated product
	json.NewEncoder(w).Encode(product)
}
