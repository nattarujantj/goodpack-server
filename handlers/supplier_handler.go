package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type SupplierHandler struct {
	repo         *repository.SupplierRepository
	purchaseRepo *repository.PurchaseRepository
}

func NewSupplierHandler(repo *repository.SupplierRepository, purchaseRepo *repository.PurchaseRepository) *SupplierHandler {
	return &SupplierHandler{
		repo:         repo,
		purchaseRepo: purchaseRepo,
	}
}

func (h *SupplierHandler) GetSuppliers(w http.ResponseWriter, r *http.Request) {
	suppliers, err := h.repo.GetAll()
	if err != nil {
		log.Printf("Error fetching suppliers: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch suppliers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suppliers)
}

func (h *SupplierHandler) GetSupplier(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid supplier ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	supplier, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, "Supplier not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(supplier)
}

func (h *SupplierHandler) CreateSupplier(w http.ResponseWriter, r *http.Request) {
	var supplierRequest models.SupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&supplierRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	supplier := supplierRequest.ToSupplier()
	if err := h.repo.Create(supplier); err != nil {
		http.Error(w, "Failed to create supplier", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(supplier)
}

func (h *SupplierHandler) UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid supplier ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	// Get existing supplier
	existingSupplier, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, "Supplier not found", http.StatusNotFound)
		return
	}

	var supplierRequest models.SupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&supplierRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update supplier
	existingSupplier.UpdateFromRequest(&supplierRequest)
	if err := h.repo.Update(id, existingSupplier); err != nil {
		http.Error(w, "Failed to update supplier", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingSupplier)
}

func (h *SupplierHandler) DeleteSupplier(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid supplier ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	if err := h.repo.Delete(id); err != nil {
		http.Error(w, "Failed to delete supplier", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SupplierHandler) GetSupplierPurchases(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	// path: /api/suppliers/{id}/purchases → [..., "suppliers", "{id}", "purchases"]
	if len(pathParts) < 5 {
		http.Error(w, "Invalid supplier ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-2]

	purchases, err := h.purchaseRepo.GetBySupplierID(context.Background(), id)
	if err != nil {
		log.Printf("Error fetching purchases for supplier %s: %v", id, err)
		http.Error(w, fmt.Sprintf("Failed to fetch purchases: %v", err), http.StatusInternalServerError)
		return
	}

	if purchases == nil {
		purchases = []*models.Purchase{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(purchases)
}

