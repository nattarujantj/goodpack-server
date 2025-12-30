package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type PurchaseOrderHandler struct {
	poRepo      *repository.PurchaseOrderRepository
	supplierRepo *repository.SupplierRepository
	productRepo  *repository.ProductRepository
}

func NewPurchaseOrderHandler(poRepo *repository.PurchaseOrderRepository, supplierRepo *repository.SupplierRepository, productRepo *repository.ProductRepository) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{
		poRepo:       poRepo,
		supplierRepo: supplierRepo,
		productRepo:  productRepo,
	}
}

func (h *PurchaseOrderHandler) GetAllPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	purchaseOrders, err := h.poRepo.GetAll()
	if err != nil {
		http.Error(w, "Failed to get purchase orders", http.StatusInternalServerError)
		return
	}

	// Populate supplier names
	for _, po := range purchaseOrders {
		if supplier, err := h.supplierRepo.GetByID(po.SupplierID); err == nil {
			po.SupplierName = supplier.CompanyName
			if len(supplier.Contacts) > 0 {
				contactName := supplier.Contacts[0].Name
				po.ContactName = &contactName
			} else if supplier.ContactName != "" {
				po.ContactName = &supplier.ContactName
			}
			if supplier.SupplierCode != "" {
				po.SupplierCode = &supplier.SupplierCode
			}
			if supplier.TaxID != "" {
				po.TaxID = &supplier.TaxID
			}
			if supplier.Address != "" {
				po.Address = &supplier.Address
			}
			if len(supplier.Contacts) > 0 && supplier.Contacts[0].Phone != "" {
				phone := supplier.Contacts[0].Phone
				po.Phone = &phone
			} else if supplier.Phone != "" {
				po.Phone = &supplier.Phone
			}
		}
	}

	json.NewEncoder(w).Encode(purchaseOrders)
}

func (h *PurchaseOrderHandler) GetPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid purchase order ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	po, err := h.poRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Purchase order not found", http.StatusNotFound)
		return
	}

	// Populate supplier information
	if supplier, err := h.supplierRepo.GetByID(po.SupplierID); err == nil {
		po.SupplierName = supplier.CompanyName
		if len(supplier.Contacts) > 0 {
			contactName := supplier.Contacts[0].Name
			po.ContactName = &contactName
		} else if supplier.ContactName != "" {
			po.ContactName = &supplier.ContactName
		}
		if supplier.SupplierCode != "" {
			po.SupplierCode = &supplier.SupplierCode
		}
		if supplier.TaxID != "" {
			po.TaxID = &supplier.TaxID
		}
		if supplier.Address != "" {
			po.Address = &supplier.Address
		}
		if len(supplier.Contacts) > 0 && supplier.Contacts[0].Phone != "" {
			phone := supplier.Contacts[0].Phone
			po.Phone = &phone
		} else if supplier.Phone != "" {
			po.Phone = &supplier.Phone
		}
	}

	json.NewEncoder(w).Encode(po)
}

func (h *PurchaseOrderHandler) CreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var poReq models.PurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&poReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate purchase order code
	lastCode, err := h.poRepo.GetLastPurchaseOrderCode(ctx)
	if err != nil {
		http.Error(w, "Failed to get last purchase order code", http.StatusInternalServerError)
		return
	}
	poCode, err := models.GeneratePurchaseOrderCode(lastCode)
	if err != nil {
		http.Error(w, "Failed to generate purchase order code", http.StatusInternalServerError)
		return
	}

	// Create purchase order
	po := poReq.ToPurchaseOrder()
	po.PurchaseOrderCode = poCode

	// Validate supplier exists
	if _, err := h.supplierRepo.GetByID(po.SupplierID); err != nil {
		http.Error(w, "Supplier not found", http.StatusBadRequest)
		return
	}

	// Validate products exist (but don't update stock or prices)
	for _, item := range po.Items {
		if _, err := h.productRepo.GetByID(ctx, item.ProductID); err != nil {
			http.Error(w, fmt.Sprintf("Product not found: %s", item.ProductID), http.StatusBadRequest)
			return
		}
	}

	// Save purchase order
	if err := h.poRepo.Create(po); err != nil {
		http.Error(w, "Failed to create purchase order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(po)
}

func (h *PurchaseOrderHandler) UpdatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid purchase order ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	var poReq models.PurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&poReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing purchase order
	existingPO, err := h.poRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Purchase order not found", http.StatusNotFound)
		return
	}

	// Update purchase order
	existingPO.UpdateFromRequest(&poReq)

	// Validate supplier exists
	if _, err := h.supplierRepo.GetByID(existingPO.SupplierID); err != nil {
		http.Error(w, "Supplier not found", http.StatusBadRequest)
		return
	}

	// Validate products exist (but don't update stock or prices)
	for _, item := range existingPO.Items {
		if _, err := h.productRepo.GetByID(ctx, item.ProductID); err != nil {
			http.Error(w, fmt.Sprintf("Product not found: %s", item.ProductID), http.StatusBadRequest)
			return
		}
	}

	// Save updated purchase order
	if err := h.poRepo.Update(id, existingPO); err != nil {
		http.Error(w, "Failed to update purchase order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingPO)
}

func (h *PurchaseOrderHandler) DeletePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid purchase order ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	if err := h.poRepo.Delete(id); err != nil {
		http.Error(w, "Failed to delete purchase order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PurchaseOrderHandler) CopyToPurchase(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid purchase order ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	// Get purchase order
	po, err := h.poRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Purchase order not found", http.StatusNotFound)
		return
	}

	// Convert to purchase request
	purchaseRequest := po.ToPurchaseRequest()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(purchaseRequest)
}

// UpdatePurchaseOrderStatus updates only the status field of a purchase order
func (h *PurchaseOrderHandler) UpdatePurchaseOrderStatus(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path: /api/purchase-orders/{id}/status
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid purchase order ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-2] // id is before "status"

	var statusReq struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&statusReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		"draft":    true,
		"sent":     true,
		"accepted": true,
		"rejected": true,
		"expired":  true,
	}
	if !validStatuses[statusReq.Status] {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	// Get existing purchase order
	po, err := h.poRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Purchase order not found", http.StatusNotFound)
		return
	}

	// Update status
	po.Status = statusReq.Status

	// Save updated purchase order
	if err := h.poRepo.Update(id, po); err != nil {
		http.Error(w, "Failed to update purchase order status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(po)
}

