package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"goodpack-server/models"
	"goodpack-server/repository"
)

type CustomerHandler struct {
	repo *repository.CustomerRepository
}

func NewCustomerHandler(repo *repository.CustomerRepository) *CustomerHandler {
	return &CustomerHandler{
		repo: repo,
	}
}

func (h *CustomerHandler) GetCustomers(w http.ResponseWriter, r *http.Request) {
	customers, err := h.repo.GetAll()
	if err != nil {
		log.Printf("Error fetching customers: %v", err)
		http.Error(w, "Failed to fetch customers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customers)
}

func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	customer, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var customerRequest models.CustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&customerRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	customer := customerRequest.ToCustomer()
	if err := h.repo.Create(customer); err != nil {
		http.Error(w, "Failed to create customer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customer)
}

func (h *CustomerHandler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	// Get existing customer
	existingCustomer, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	var customerRequest models.CustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&customerRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update customer
	existingCustomer.UpdateFromRequest(&customerRequest)
	if err := h.repo.Update(id, existingCustomer); err != nil {
		http.Error(w, "Failed to update customer", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingCustomer)
}

func (h *CustomerHandler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	if err := h.repo.Delete(id); err != nil {
		http.Error(w, "Failed to delete customer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
