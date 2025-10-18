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

type QuotationHandler struct {
	quotationRepo *repository.QuotationRepository
	customerRepo  *repository.CustomerRepository
	productRepo   *repository.ProductRepository
}

func NewQuotationHandler(quotationRepo *repository.QuotationRepository, customerRepo *repository.CustomerRepository, productRepo *repository.ProductRepository) *QuotationHandler {
	return &QuotationHandler{
		quotationRepo: quotationRepo,
		customerRepo:  customerRepo,
		productRepo:   productRepo,
	}
}

func (h *QuotationHandler) GetAllQuotations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	quotations, err := h.quotationRepo.GetAll()
	if err != nil {
		http.Error(w, "Failed to get quotations", http.StatusInternalServerError)
		return
	}

	// Populate customer names
	for _, quotation := range quotations {
		if customer, err := h.customerRepo.GetByID(quotation.CustomerID); err == nil {
			quotation.CustomerName = customer.CompanyName
			if customer.ContactName != "" {
				quotation.ContactName = &customer.ContactName
			}
			if customer.CustomerCode != "" {
				quotation.CustomerCode = &customer.CustomerCode
			}
			if customer.TaxID != "" {
				quotation.TaxID = &customer.TaxID
			}
			if customer.Address != "" {
				quotation.Address = &customer.Address
			}
			if customer.Phone != "" {
				quotation.Phone = &customer.Phone
			}
		}
	}

	json.NewEncoder(w).Encode(quotations)
}

func (h *QuotationHandler) GetQuotation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid quotation ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	quotation, err := h.quotationRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Quotation not found", http.StatusNotFound)
		return
	}

	// Populate customer information
	if customer, err := h.customerRepo.GetByID(quotation.CustomerID); err == nil {
		quotation.CustomerName = customer.CompanyName
		if customer.ContactName != "" {
			quotation.ContactName = &customer.ContactName
		}
		if customer.CustomerCode != "" {
			quotation.CustomerCode = &customer.CustomerCode
		}
		if customer.TaxID != "" {
			quotation.TaxID = &customer.TaxID
		}
		if customer.Address != "" {
			quotation.Address = &customer.Address
		}
		if customer.Phone != "" {
			quotation.Phone = &customer.Phone
		}
	}

	json.NewEncoder(w).Encode(quotation)
}

func (h *QuotationHandler) CreateQuotation(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var quotationReq models.QuotationRequest
	if err := json.NewDecoder(r.Body).Decode(&quotationReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate quotation code
	lastCode, err := h.quotationRepo.GetLastQuotationCode(ctx)
	if err != nil {
		http.Error(w, "Failed to get last quotation code", http.StatusInternalServerError)
		return
	}
	quotationCode, err := models.GenerateQuotationCode(lastCode)
	if err != nil {
		http.Error(w, "Failed to generate quotation code", http.StatusInternalServerError)
		return
	}

	// Create quotation
	quotation := quotationReq.ToQuotation()
	quotation.QuotationCode = quotationCode

	// Validate customer exists
	if _, err := h.customerRepo.GetByID(quotation.CustomerID); err != nil {
		http.Error(w, "Customer not found", http.StatusBadRequest)
		return
	}

	// Validate products exist (but don't update stock or prices)
	for _, item := range quotation.Items {
		if _, err := h.productRepo.GetByID(ctx, item.ProductID); err != nil {
			http.Error(w, fmt.Sprintf("Product not found: %s", item.ProductID), http.StatusBadRequest)
			return
		}
	}

	// Save quotation
	if err := h.quotationRepo.Create(quotation); err != nil {
		http.Error(w, "Failed to create quotation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(quotation)
}

func (h *QuotationHandler) UpdateQuotation(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid quotation ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	var quotationReq models.QuotationRequest
	if err := json.NewDecoder(r.Body).Decode(&quotationReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing quotation
	existingQuotation, err := h.quotationRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Quotation not found", http.StatusNotFound)
		return
	}

	// Update quotation
	existingQuotation.UpdateFromRequest(&quotationReq)

	// Validate customer exists
	if _, err := h.customerRepo.GetByID(existingQuotation.CustomerID); err != nil {
		http.Error(w, "Customer not found", http.StatusBadRequest)
		return
	}

	// Validate products exist (but don't update stock or prices)
	for _, item := range existingQuotation.Items {
		if _, err := h.productRepo.GetByID(ctx, item.ProductID); err != nil {
			http.Error(w, fmt.Sprintf("Product not found: %s", item.ProductID), http.StatusBadRequest)
			return
		}
	}

	// Save updated quotation
	if err := h.quotationRepo.Update(id, existingQuotation); err != nil {
		http.Error(w, "Failed to update quotation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingQuotation)
}

func (h *QuotationHandler) DeleteQuotation(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid quotation ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	if err := h.quotationRepo.Delete(id); err != nil {
		http.Error(w, "Failed to delete quotation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *QuotationHandler) CopyToSale(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid quotation ID", http.StatusBadRequest)
		return
	}
	id := pathParts[len(pathParts)-1]

	// Get quotation
	quotation, err := h.quotationRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Quotation not found", http.StatusNotFound)
		return
	}

	// Convert to sale request
	saleRequest := quotation.ToSaleRequest()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saleRequest)
}
