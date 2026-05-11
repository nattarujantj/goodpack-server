package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"goodpack-server/middleware"
	"goodpack-server/models"
	"goodpack-server/services"
)

type InventorySnapshotHandler struct {
	snapshotService *services.InventorySnapshotService
}

func NewInventorySnapshotHandler(snapshotService *services.InventorySnapshotService) *InventorySnapshotHandler {
	return &InventorySnapshotHandler{snapshotService: snapshotService}
}

type snapshotCreateRequest struct {
	Month int `json:"month"`
	Year  int `json:"year"`
}

// CreateSnapshot handles POST /api/inventory-snapshots — manual trigger
func (h *InventorySnapshotHandler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req snapshotCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendSnapshotError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Month < 1 || req.Month > 12 {
		sendSnapshotError(w, "Invalid month (1-12)", http.StatusBadRequest)
		return
	}
	if req.Year < 2020 || req.Year > 2100 {
		sendSnapshotError(w, "Invalid year", http.StatusBadRequest)
		return
	}

	createdBy := middleware.GetUserID(r)
	if createdBy == "" {
		createdBy = "unknown"
	}

	log.Printf("📸 Manual snapshot requested: month=%d year=%d by=%s", req.Month, req.Year, createdBy)

	snap, err := h.snapshotService.TakeSnapshot(r.Context(), req.Month, req.Year, createdBy, true)
	if err != nil {
		log.Printf("❌ Manual snapshot failed: %v", err)
		sendSnapshotError(w, "Failed to create snapshot: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Manual snapshot created: %d products, month=%d year=%d", snap.TotalProducts, snap.Month, snap.Year)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(snap)
}

// ListSnapshots handles GET /api/inventory-snapshots
func (h *InventorySnapshotHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := h.snapshotService.ListSnapshots(r.Context())
	if err != nil {
		log.Printf("❌ List snapshots failed: %v", err)
		sendSnapshotError(w, "Failed to list snapshots", http.StatusInternalServerError)
		return
	}
	if snapshots == nil {
		snapshots = []*models.InventorySnapshot{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshots)
}

// GetSnapshot handles GET /api/inventory-snapshots/{year}/{month}
func (h *InventorySnapshotHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year, err := strconv.Atoi(vars["year"])
	if err != nil {
		sendSnapshotError(w, "Invalid year", http.StatusBadRequest)
		return
	}
	month, err := strconv.Atoi(vars["month"])
	if err != nil {
		sendSnapshotError(w, "Invalid month", http.StatusBadRequest)
		return
	}

	snap, err := h.snapshotService.GetSnapshot(r.Context(), month, year)
	if err != nil {
		sendSnapshotError(w, "Failed to get snapshot", http.StatusInternalServerError)
		return
	}
	if snap == nil {
		sendSnapshotError(w, "Snapshot not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

// UpdateSnapshot handles PUT /api/inventory-snapshots/{year}/{month}
func (h *InventorySnapshotHandler) UpdateSnapshot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	year, err := strconv.Atoi(vars["year"])
	if err != nil {
		sendSnapshotError(w, "Invalid year", http.StatusBadRequest)
		return
	}
	month, err := strconv.Atoi(vars["month"])
	if err != nil {
		sendSnapshotError(w, "Invalid month", http.StatusBadRequest)
		return
	}

	var req models.SnapshotUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendSnapshotError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdBy := middleware.GetUserID(r)
	if createdBy == "" {
		createdBy = "unknown"
	}

	log.Printf("✏️ Snapshot update: month=%d year=%d by=%s, products=%d", month, year, createdBy, len(req.Products))

	snap, err := h.snapshotService.UpdateSnapshot(r.Context(), month, year, req.Products, createdBy)
	if err != nil {
		log.Printf("❌ Snapshot update failed: %v", err)
		sendSnapshotError(w, "Failed to update snapshot: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

func sendSnapshotError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
