package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

type InternationalImportHandler struct {
	importRepo          *repository.InternationalImportRepository
	supplierRepo        *repository.SupplierRepository
	shippingRepo        *repository.ShippingCompanyRepository
	productRepo         *repository.ProductRepository
	purchaseRepo        *repository.PurchaseRepository
	stockAdjustmentRepo *repository.StockAdjustmentRepository
}

func NewInternationalImportHandler(
	importRepo *repository.InternationalImportRepository,
	supplierRepo *repository.SupplierRepository,
	shippingRepo *repository.ShippingCompanyRepository,
	productRepo *repository.ProductRepository,
	purchaseRepo *repository.PurchaseRepository,
	stockAdjustmentRepo *repository.StockAdjustmentRepository,
) *InternationalImportHandler {
	return &InternationalImportHandler{
		importRepo:          importRepo,
		supplierRepo:        supplierRepo,
		shippingRepo:        shippingRepo,
		productRepo:         productRepo,
		purchaseRepo:        purchaseRepo,
		stockAdjustmentRepo: stockAdjustmentRepo,
	}
}

func (h *InternationalImportHandler) enrichWithNames(imp *models.InternationalImport) {
	if supplier, err := h.supplierRepo.GetByID(imp.SupplierID); err == nil {
		imp.SupplierName = supplier.CompanyName
		if imp.SupplierName == "" {
			imp.SupplierName = supplier.ContactName
		}
	}
	if shipping, err := h.shippingRepo.GetByID(imp.ShippingCompanyID); err == nil {
		imp.ShippingCompanyName = shipping.Name
	}
}

func (h *InternationalImportHandler) generateImportCode(ctx context.Context) (string, error) {
	now := utils.NowInThailand()
	beYear := now.Year() + 543
	dateStr := fmt.Sprintf("%02d%02d", beYear%100, int(now.Month()))
	prefix := "IMP-" + dateStr + "-"

	nextSeq, err := h.importRepo.GetNextSequenceNumber(ctx, "IMP-")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, nextSeq), nil
}

func (h *InternationalImportHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	imports, err := h.importRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch international imports", http.StatusInternalServerError)
		return
	}
	if imports == nil {
		imports = []*models.InternationalImport{}
	}

	for i := range imports {
		h.enrichWithNames(imports[i])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imports)
}

func (h *InternationalImportHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	imp, err := h.importRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "International import not found", http.StatusNotFound)
		return
	}

	h.enrichWithNames(imp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imp)
}

func (h *InternationalImportHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req models.InternationalImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if _, err := h.supplierRepo.GetByID(req.SupplierID); err != nil {
		http.Error(w, "Supplier not found", http.StatusBadRequest)
		return
	}

	imp := req.ToInternationalImport()

	importCode, err := h.generateImportCode(ctx)
	if err != nil {
		http.Error(w, "Failed to generate import code", http.StatusInternalServerError)
		return
	}
	imp.ImportCode = importCode

	h.enrichWithNames(imp)

	if err := h.importRepo.Create(ctx, imp); err != nil {
		http.Error(w, "Failed to create international import", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(imp)
}

func (h *InternationalImportHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	existing, err := h.importRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "International import not found", http.StatusNotFound)
		return
	}

	var req models.InternationalImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existing.UpdateFromRequest(&req)
	h.enrichWithNames(existing)

	if err := h.importRepo.Update(ctx, id, existing); err != nil {
		http.Error(w, "Failed to update international import", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *InternationalImportHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	if err := h.importRepo.Delete(ctx, id); err != nil {
		http.Error(w, "Failed to delete international import", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// CreatePurchaseFromImport creates a Purchase record from an international import.
func (h *InternationalImportHandler) CreatePurchaseFromImport(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pathParts := strings.Split(r.URL.Path, "/")
	// URL: /api/international-imports/{id}/create-purchase
	if len(pathParts) < 5 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-2]

	imp, err := h.importRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "International import not found", http.StatusNotFound)
		return
	}

	if imp.Status == "purchased" {
		http.Error(w, "Purchase already created for this import", http.StatusBadRequest)
		return
	}

	purchaseReq := imp.ToPurchaseRequest()

	supplier, err := h.supplierRepo.GetByID(purchaseReq.SupplierID)
	if err != nil {
		http.Error(w, "Supplier not found", http.StatusBadRequest)
		return
	}

	purchase := purchaseReq.ToPurchase()
	purchase.SupplierName = supplier.CompanyName
	if purchase.SupplierName == "" {
		purchase.SupplierName = supplier.ContactName
	}
	purchase.ContactName = &supplier.ContactName

	purchaseHandler := NewPurchaseHandler(h.purchaseRepo, h.supplierRepo, h.productRepo, h.stockAdjustmentRepo)
	purchaseCode, err := purchaseHandler.generatePurchaseID(ctx, purchase.IsVAT)
	if err != nil {
		http.Error(w, "Failed to generate purchase code", http.StatusInternalServerError)
		return
	}
	purchase.PurchaseCode = purchaseCode

	if err := h.purchaseRepo.Create(ctx, purchase); err != nil {
		http.Error(w, "Failed to create purchase", http.StatusInternalServerError)
		return
	}

	purchaseID := purchase.ID.Hex()
	imp.Status = "purchased"
	imp.PurchaseID = &purchaseID
	imp.UpdatedAt = utils.NowInThailand()

	if err := h.importRepo.Update(ctx, id, imp); err != nil {
		http.Error(w, "Failed to update import status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"import":   imp,
		"purchase": purchase,
	})
}
