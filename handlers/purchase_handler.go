package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

type PurchaseHandler struct {
	purchaseRepo        *repository.PurchaseRepository
	supplierRepo        *repository.SupplierRepository
	productRepo         *repository.ProductRepository
	stockAdjustmentRepo *repository.StockAdjustmentRepository
}

func NewPurchaseHandler(purchaseRepo *repository.PurchaseRepository, supplierRepo *repository.SupplierRepository, productRepo *repository.ProductRepository, stockAdjustmentRepo *repository.StockAdjustmentRepository) *PurchaseHandler {
	return &PurchaseHandler{
		purchaseRepo:        purchaseRepo,
		supplierRepo:        supplierRepo,
		productRepo:         productRepo,
		stockAdjustmentRepo: stockAdjustmentRepo,
	}
}

// enrichPurchaseWithSupplierData enriches a purchase with supplier data
func (h *PurchaseHandler) enrichPurchaseWithSupplierData(purchase *models.Purchase) {
	supplier, err := h.supplierRepo.GetByID(purchase.SupplierID)
	if err == nil {
		// Update purchase with supplier data
		purchase.SupplierName = supplier.CompanyName
		if purchase.SupplierName == "" {
			purchase.SupplierName = supplier.ContactName
		}
		purchase.ContactName = &supplier.ContactName
		purchase.SupplierCode = &supplier.SupplierCode
		purchase.TaxID = &supplier.TaxID
		purchase.Address = &supplier.Address
		purchase.Phone = &supplier.Phone
	}
}

// generatePurchaseID generates a unique purchase ID based on VAT status.
// Sequence continues across months (e.g. PUR-VAT-6901-0005 -> PUR-VAT-6902-0006).
func (h *PurchaseHandler) generatePurchaseID(ctx context.Context, isVAT bool) (string, error) {
	now := utils.NowInThailand()
	beYear := now.Year() + 543
	dateStr := fmt.Sprintf("%02d%02d", beYear%100, int(now.Month())) // YYMM format

	// Base prefix for sequence lookup (no month - continue across months)
	var basePrefix string
	if isVAT {
		basePrefix = "PUR-VAT-"
	} else {
		basePrefix = "PUR-NV-"
	}

	nextSeq, err := h.purchaseRepo.GetNextSequenceNumber(ctx, basePrefix)
	if err != nil {
		return "", err
	}

	seqStr := fmt.Sprintf("%04d", nextSeq)
	fullPrefix := fmt.Sprintf("PUR-VAT-%s", dateStr)
	if !isVAT {
		fullPrefix = fmt.Sprintf("PUR-NV-%s", dateStr)
	}
	return fullPrefix + "-" + seqStr, nil
}

func (h *PurchaseHandler) GetPurchases(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	purchases, err := h.purchaseRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch purchases", http.StatusInternalServerError)
		return
	}

	// Enrich purchases with supplier data
	for i := range purchases {
		h.enrichPurchaseWithSupplierData(purchases[i])
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

	// Enrich purchase with supplier data
	h.enrichPurchaseWithSupplierData(purchase)

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

	// Get supplier name
	supplier, err := h.supplierRepo.GetByID(purchaseRequest.SupplierID)
	if err != nil {
		http.Error(w, "Supplier not found", http.StatusBadRequest)
		return
	}

	purchase := purchaseRequest.ToPurchase()
	purchase.SupplierName = supplier.CompanyName
	if purchase.SupplierName == "" {
		purchase.SupplierName = supplier.ContactName
	}
	purchase.ContactName = &supplier.ContactName

	var purchaseCode string
	if purchaseRequest.PurchaseCode != nil && strings.TrimSpace(*purchaseRequest.PurchaseCode) != "" {
		purchaseCode = strings.TrimSpace(*purchaseRequest.PurchaseCode)
		existing, err := h.purchaseRepo.GetByPurchaseCode(ctx, purchaseCode)
		if err == nil && existing != nil {
			http.Error(w, "Purchase code already exists", http.StatusBadRequest)
			return
		}
	} else {
		var err error
		purchaseCode, err = h.generatePurchaseID(ctx, purchase.IsVAT)
		if err != nil {
			http.Error(w, "Failed to generate purchase code", http.StatusInternalServerError)
			return
		}
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

	// If purchaseCode is being changed, validate uniqueness
	if purchaseRequest.PurchaseCode != nil && strings.TrimSpace(*purchaseRequest.PurchaseCode) != "" {
		newCode := strings.TrimSpace(*purchaseRequest.PurchaseCode)
		if newCode != existingPurchase.PurchaseCode {
			other, err := h.purchaseRepo.GetByPurchaseCode(ctx, newCode)
			if err == nil && other != nil && other.ID.Hex() != id {
				http.Error(w, "Purchase code already exists", http.StatusBadRequest)
				return
			}
		}
	}

	// Get supplier name
	supplier, err := h.supplierRepo.GetByID(purchaseRequest.SupplierID)
	if err != nil {
		http.Error(w, "Supplier not found", http.StatusBadRequest)
		return
	}

	// Rollback price updates and preform stock from old purchase items
	for _, oldItem := range existingPurchase.Items {
		product, err := h.productRepo.GetByID(ctx, oldItem.ProductID)
		if err == nil {
			// Calculate effective unit price (including preform cost if any)
			effectiveUnitPrice := oldItem.UnitPrice
			if oldItem.PreformUnitPrice != nil {
				effectiveUnitPrice += *oldItem.PreformUnitPrice
			}
			product.RollbackPriceUpdate(effectiveUnitPrice, existingPurchase.IsVAT, true, oldItem.Quantity)
			h.productRepo.Update(ctx, oldItem.ProductID, product)
		}

		// Rollback preform stock if used
		if oldItem.PreformProductID != nil && *oldItem.PreformProductID != "" {
			preformProduct, err := h.productRepo.GetByID(ctx, *oldItem.PreformProductID)
			if err == nil {
				// Determine stock type based on VAT status
				var stockType models.StockType
				if existingPurchase.IsVAT {
					stockType = models.StockTypeVAT
				} else {
					stockType = models.StockTypeNonVAT
				}
				// Restore preform stock (rollback the deduction)
				ApplyStockAdjustment(preformProduct, models.AdjustmentTypeAdd, stockType, oldItem.Quantity)
				h.productRepo.Update(ctx, *oldItem.PreformProductID, preformProduct)
			}
		}
	}

	// Update purchase
	existingPurchase.UpdateFromRequest(&purchaseRequest)
	existingPurchase.SupplierName = supplier.CompanyName
	if existingPurchase.SupplierName == "" {
		existingPurchase.SupplierName = supplier.ContactName
	}

	if err := h.purchaseRepo.Update(ctx, id, existingPurchase); err != nil {
		http.Error(w, "Failed to update purchase", http.StatusInternalServerError)
		return
	}

	// Update product prices and stock with new data
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

	// Get existing purchase to rollback price updates
	existingPurchase, err := h.purchaseRepo.GetByID(ctx, id)
	if err == nil {
		// Rollback price updates and preform stock from purchase items
		for _, item := range existingPurchase.Items {
			product, err := h.productRepo.GetByID(ctx, item.ProductID)
			if err == nil {
				// Calculate effective unit price (including preform cost if any)
				effectiveUnitPrice := item.UnitPrice
				if item.PreformUnitPrice != nil {
					effectiveUnitPrice += *item.PreformUnitPrice
				}
				product.RollbackPriceUpdate(effectiveUnitPrice, existingPurchase.IsVAT, true, item.Quantity)
				h.productRepo.Update(ctx, item.ProductID, product)
			}

			// Rollback preform stock if used
			if item.PreformProductID != nil && *item.PreformProductID != "" {
				preformProduct, err := h.productRepo.GetByID(ctx, *item.PreformProductID)
				if err == nil {
					// Determine stock type based on VAT status
					var stockType models.StockType
					if existingPurchase.IsVAT {
						stockType = models.StockTypeVAT
					} else {
						stockType = models.StockTypeNonVAT
					}
					// Restore preform stock (rollback the deduction)
					ApplyStockAdjustment(preformProduct, models.AdjustmentTypeAdd, stockType, item.Quantity)
					h.productRepo.Update(ctx, *item.PreformProductID, preformProduct)
				}
			}
		}
	}

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

		// Calculate effective unit price (including preform cost if any)
		effectiveUnitPrice := item.UnitPrice
		if item.PreformUnitPrice != nil {
			effectiveUnitPrice += *item.PreformUnitPrice
		}

		// Update purchase price using new UpdatePrice method with quantity
		product.UpdatePrice(effectiveUnitPrice, purchase.IsVAT, true, item.Quantity) // true = isPurchase

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

		// If preform is used, deduct preform stock
		if item.PreformProductID != nil && *item.PreformProductID != "" {
			preformProduct, err := h.productRepo.GetByID(ctx, *item.PreformProductID)
			if err == nil {
				// Determine stock type based on VAT status (same as the item being purchased)
				var preformStockType models.StockType
				if purchase.IsVAT {
					preformStockType = models.StockTypeVAT
				} else {
					preformStockType = models.StockTypeNonVAT
				}
				// Deduct preform stock according to VAT/Non-VAT status
				ApplyStockAdjustment(preformProduct, models.AdjustmentTypeReduce, preformStockType, item.Quantity)

				// Save updated preform product
				if err := h.productRepo.Update(ctx, *item.PreformProductID, preformProduct); err != nil {
					fmt.Printf("Warning: Failed to update preform product stock: %v\n", err)
				} else {
					// Record preform stock change in history
					preformNotes := fmt.Sprintf("ใช้ในรายการซื้อ %s", purchaseCode)
					if err := RecordStockChange(
						ctx,
						h.stockAdjustmentRepo,
						preformProduct,
						models.SourceTypePurchase,
						&purchaseID,
						&purchaseCode,
						models.AdjustmentTypeReduce,
						preformStockType,
						item.Quantity,
						&preformNotes,
					); err != nil {
						fmt.Printf("Warning: Failed to record preform stock change history: %v\n", err)
					}
				}
			}
		}
	}

	return nil
}
