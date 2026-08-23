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

type FCLShipmentHandler struct {
	fclRepo      *repository.FCLShipmentRepository
	importRepo   *repository.InternationalImportRepository
	shippingRepo *repository.ShippingCompanyRepository
	supplierRepo *repository.SupplierRepository
}

func NewFCLShipmentHandler(
	fclRepo *repository.FCLShipmentRepository,
	importRepo *repository.InternationalImportRepository,
	shippingRepo *repository.ShippingCompanyRepository,
	supplierRepo *repository.SupplierRepository,
) *FCLShipmentHandler {
	return &FCLShipmentHandler{
		fclRepo:      fclRepo,
		importRepo:   importRepo,
		shippingRepo: shippingRepo,
		supplierRepo: supplierRepo,
	}
}

// recomputeFCLContainer refreshes a container's total CBM / linked count and, while the
// container is open, re-allocates shipping cost across every linked import by CBM share.
// Once the container is closed, linked imports keep their locked snapshot values.
// This is the single place the CBM-average is applied, so it stays correct no matter the
// order in which factories are entered or edited.
func recomputeFCLContainer(
	ctx context.Context,
	importRepo *repository.InternationalImportRepository,
	fclRepo *repository.FCLShipmentRepository,
	shipmentID string,
) error {
	shipment, err := fclRepo.GetByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	imports, err := importRepo.GetByFCLShipmentID(ctx, shipmentID)
	if err != nil {
		return err
	}

	containerCBM := 0.0
	for _, imp := range imports {
		containerCBM += imp.RecalcItemCBM()
	}

	shipment.TotalCBM = containerCBM
	shipment.LinkedImportCount = len(imports)
	shipment.UpdatedAt = utils.NowInThailand()
	if err := fclRepo.Update(ctx, shipmentID, shipment); err != nil {
		return err
	}

	if shipment.Status == "open" {
		for _, imp := range imports {
			imp.RecalculateForFCLContainer(shipment.TotalCostThb, containerCBM)
			if err := importRepo.Update(ctx, imp.ID.Hex(), imp); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *FCLShipmentHandler) enrich(s *models.FCLShipment) {
	if s.ShippingCompanyID != "" {
		if sc, err := h.shippingRepo.GetByID(s.ShippingCompanyID); err == nil {
			s.ShippingCompanyName = sc.Name
		}
	}
}

func (h *FCLShipmentHandler) generateFCLCode(ctx context.Context) (string, error) {
	now := utils.NowInThailand()
	beYear := now.Year() + 543
	dateStr := fmt.Sprintf("%02d%02d", beYear%100, int(now.Month()))
	prefix := "FCL-" + dateStr + "-"

	nextSeq, err := h.fclRepo.GetNextSequenceNumber(ctx, "FCL-")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%04d", prefix, nextSeq), nil
}

func (h *FCLShipmentHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	shipments, err := h.fclRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch FCL shipments", http.StatusInternalServerError)
		return
	}
	if shipments == nil {
		shipments = []*models.FCLShipment{}
	}
	for i := range shipments {
		h.enrich(shipments[i])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shipments)
}

func (h *FCLShipmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := lastPathSegment(r.URL.Path)

	shipment, err := h.fclRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "FCL shipment not found", http.StatusNotFound)
		return
	}
	h.enrich(shipment)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shipment)
}

// GetImports returns every import linked to this container (the factories in the container).
// URL: GET /api/fcl-shipments/{id}/imports
func (h *FCLShipmentHandler) GetImports(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-2]

	imports, err := h.importRepo.GetByFCLShipmentID(ctx, id)
	if err != nil {
		http.Error(w, "Failed to fetch imports", http.StatusInternalServerError)
		return
	}
	if imports == nil {
		imports = []*models.InternationalImport{}
	}
	for _, imp := range imports {
		if imp.SupplierID != "" && imp.SupplierName == "" {
			if supplier, err := h.supplierRepo.GetByID(imp.SupplierID); err == nil {
				imp.SupplierName = supplier.CompanyName
				if imp.SupplierName == "" {
					imp.SupplierName = supplier.ContactName
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imports)
}

func (h *FCLShipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req models.FCLShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	shipment := req.ToFCLShipment()

	fclCode, err := h.generateFCLCode(ctx)
	if err != nil {
		http.Error(w, "Failed to generate FCL code", http.StatusInternalServerError)
		return
	}
	shipment.FCLCode = fclCode
	h.enrich(shipment)

	if err := h.fclRepo.Create(ctx, shipment); err != nil {
		http.Error(w, "Failed to create FCL shipment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shipment)
}

func (h *FCLShipmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := lastPathSegment(r.URL.Path)

	existing, err := h.fclRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "FCL shipment not found", http.StatusNotFound)
		return
	}
	if existing.Status == "closed" {
		http.Error(w, "Container is closed. Reopen it before editing.", http.StatusBadRequest)
		return
	}

	var req models.FCLShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existing.UpdateFromRequest(&req)
	if err := h.fclRepo.Update(ctx, id, existing); err != nil {
		http.Error(w, "Failed to update FCL shipment", http.StatusInternalServerError)
		return
	}

	// Cost lines changed → re-allocate to every linked import.
	if err := recomputeFCLContainer(ctx, h.importRepo, h.fclRepo, id); err != nil {
		http.Error(w, "Failed to recompute container", http.StatusInternalServerError)
		return
	}

	updated, err := h.fclRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "FCL shipment not found", http.StatusNotFound)
		return
	}
	h.enrich(updated)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *FCLShipmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	id := lastPathSegment(r.URL.Path)

	imports, err := h.importRepo.GetByFCLShipmentID(ctx, id)
	if err != nil {
		http.Error(w, "Failed to check linked imports", http.StatusInternalServerError)
		return
	}
	if len(imports) > 0 {
		http.Error(w, "Cannot delete a container that still has linked imports", http.StatusBadRequest)
		return
	}

	if err := h.fclRepo.Delete(ctx, id); err != nil {
		http.Error(w, "Failed to delete FCL shipment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Close finalizes the container: it recomputes the allocation one last time, then locks it.
// URL: PATCH /api/fcl-shipments/{id}/close
func (h *FCLShipmentHandler) Close(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-2]

	shipment, err := h.fclRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "FCL shipment not found", http.StatusNotFound)
		return
	}
	if shipment.Status == "closed" {
		http.Error(w, "Container is already closed", http.StatusBadRequest)
		return
	}

	// Final allocation while still open, then lock.
	if err := recomputeFCLContainer(ctx, h.importRepo, h.fclRepo, id); err != nil {
		http.Error(w, "Failed to recompute container", http.StatusInternalServerError)
		return
	}

	shipment, err = h.fclRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "FCL shipment not found", http.StatusNotFound)
		return
	}
	shipment.Status = "closed"
	shipment.UpdatedAt = utils.NowInThailand()
	if err := h.fclRepo.Update(ctx, id, shipment); err != nil {
		http.Error(w, "Failed to close container", http.StatusInternalServerError)
		return
	}
	h.enrich(shipment)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shipment)
}

// Reopen unlocks a closed container and re-allocates the cost live again.
// URL: PATCH /api/fcl-shipments/{id}/reopen
func (h *FCLShipmentHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-2]

	shipment, err := h.fclRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "FCL shipment not found", http.StatusNotFound)
		return
	}
	if shipment.Status != "closed" {
		http.Error(w, "Only a closed container can be reopened", http.StatusBadRequest)
		return
	}

	shipment.Status = "open"
	shipment.UpdatedAt = utils.NowInThailand()
	if err := h.fclRepo.Update(ctx, id, shipment); err != nil {
		http.Error(w, "Failed to reopen container", http.StatusInternalServerError)
		return
	}

	if err := recomputeFCLContainer(ctx, h.importRepo, h.fclRepo, id); err != nil {
		http.Error(w, "Failed to recompute container", http.StatusInternalServerError)
		return
	}

	updated, err := h.fclRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "FCL shipment not found", http.StatusNotFound)
		return
	}
	h.enrich(updated)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// lastPathSegment returns the final segment of a URL path (the {id}).
func lastPathSegment(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	return parts[len(parts)-1]
}
