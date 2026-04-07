package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type ShippingCompanyHandler struct {
	repo *repository.ShippingCompanyRepository
}

func NewShippingCompanyHandler(repo *repository.ShippingCompanyRepository) *ShippingCompanyHandler {
	return &ShippingCompanyHandler{repo: repo}
}

func (h *ShippingCompanyHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	companies, err := h.repo.GetAll()
	if err != nil {
		http.Error(w, "Failed to fetch shipping companies", http.StatusInternalServerError)
		return
	}
	if companies == nil {
		companies = []*models.ShippingCompany{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(companies)
}

func (h *ShippingCompanyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.ShippingCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	company := req.ToShippingCompany()
	if err := h.repo.Create(company); err != nil {
		http.Error(w, "Failed to create shipping company", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(company)
}

func (h *ShippingCompanyHandler) Update(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	existing, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, "Shipping company not found", http.StatusNotFound)
		return
	}

	var req models.ShippingCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existing.UpdateFromRequest(&req)
	if err := h.repo.Update(id, existing); err != nil {
		http.Error(w, "Failed to update shipping company", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *ShippingCompanyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	if err := h.repo.Delete(id); err != nil {
		http.Error(w, "Failed to delete shipping company", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
