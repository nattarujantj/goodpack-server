package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

type ExpenseHandler struct {
	expenseRepo *repository.ExpenseRepository
}

func NewExpenseHandler(expenseRepo *repository.ExpenseRepository) *ExpenseHandler {
	return &ExpenseHandler{
		expenseRepo: expenseRepo,
	}
}

func (h *ExpenseHandler) GetExpenses(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	expenses, err := h.expenseRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if expenses == nil {
		expenses = []*models.Expense{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expenses)
}

func (h *ExpenseHandler) GetExpense(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	expense, err := h.expenseRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expense)
}

func (h *ExpenseHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var req models.ExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Category == "" {
		http.Error(w, "Category is required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		expenseDate = utils.NowInThailand()
	}

	now := utils.NowInThailand()
	expense := &models.Expense{
		Category:    req.Category,
		Description: req.Description,
		Amount:      req.Amount,
		ExpenseDate: expenseDate,
		Notes:       req.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ctx := context.Background()
	if err := h.expenseRepo.Create(ctx, expense); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(expense)
}

func (h *ExpenseHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	existing, err := h.expenseRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	var req models.ExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Category == "" {
		http.Error(w, "Category is required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		expenseDate = existing.ExpenseDate
	}

	existing.Category = req.Category
	existing.Description = req.Description
	existing.Amount = req.Amount
	existing.ExpenseDate = expenseDate
	existing.Notes = req.Notes
	existing.UpdatedAt = utils.NowInThailand()

	if err := h.expenseRepo.Update(ctx, id, existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *ExpenseHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	if err := h.expenseRepo.Delete(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Expense deleted successfully"})
}

func (h *ExpenseHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.DefaultExpenseCategories)
}
