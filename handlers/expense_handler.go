package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"

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

	// Best-effort cleanup of any uploaded attachment files
	if expense, err := h.expenseRepo.GetByID(ctx, id); err == nil {
		for _, a := range expense.Attachments {
			filePath := strings.TrimPrefix(a.FileURL, "/")
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: Failed to delete attachment file %s: %v\n", filePath, err)
			}
		}
	}

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

// UploadAttachment handles uploading a bill/receipt file (PDF or image) for an expense.
func (h *ExpenseHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	expense, err := h.expenseRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	// Parse multipart form with 15MB max memory
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Limit file size to 10MB
	if header.Size > 10*1024*1024 {
		http.Error(w, "File size too large. Maximum size is 10MB", http.StatusBadRequest)
		return
	}

	// Detect file type by reading the file signature
	sig := make([]byte, 12)
	if _, err := file.Read(sig); err != nil && err != io.EOF {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	file.Seek(0, 0)

	fileType, ok := detectAttachmentType(sig)
	if !ok {
		http.Error(w, "Invalid file type. Only PDF, JPEG, PNG, GIF, and WebP are allowed", http.StatusBadRequest)
		return
	}

	// Ensure upload directory exists
	uploadDir := "uploads/expenses"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Generate a unique filename, preserving a sensible extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		if fileType == "pdf" {
			ext = ".pdf"
		} else {
			ext = ".jpg"
		}
	}
	attachmentID := primitive.NewObjectID().Hex()
	filename := fmt.Sprintf("%s_%s%s", id, attachmentID, ext)
	filePath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(filePath)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	attachment := models.ExpenseAttachment{
		ID:         attachmentID,
		FileName:   header.Filename,
		FileURL:    fmt.Sprintf("/uploads/expenses/%s", filename),
		FileType:   fileType,
		Size:       header.Size,
		UploadedAt: utils.NowInThailand(),
	}

	expense.Attachments = append(expense.Attachments, attachment)
	expense.UpdatedAt = utils.NowInThailand()

	if err := h.expenseRepo.Update(ctx, id, expense); err != nil {
		os.Remove(filePath)
		http.Error(w, "Failed to update expense", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(attachment)
}

// DeleteAttachment removes a single attachment from an expense and deletes the file.
func (h *ExpenseHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	attachmentID := vars["attachmentId"]

	ctx := context.Background()
	expense, err := h.expenseRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return
	}

	index := -1
	for i, a := range expense.Attachments {
		if a.ID == attachmentID {
			index = i
			break
		}
	}
	if index == -1 {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	target := expense.Attachments[index]

	// Remove the file from disk (best effort)
	filePath := strings.TrimPrefix(target.FileURL, "/")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: Failed to delete attachment file %s: %v\n", filePath, err)
	}

	expense.Attachments = append(expense.Attachments[:index], expense.Attachments[index+1:]...)
	expense.UpdatedAt = utils.NowInThailand()

	if err := h.expenseRepo.Update(ctx, id, expense); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Attachment deleted successfully"})
}

// detectAttachmentType inspects the leading bytes of a file and returns
// ("pdf" | "image", true) for supported types, or ("", false) otherwise.
func detectAttachmentType(sig []byte) (string, bool) {
	if len(sig) >= 4 {
		// PDF signature: 25 50 44 46  (%PDF)
		if sig[0] == 0x25 && sig[1] == 0x50 && sig[2] == 0x44 && sig[3] == 0x46 {
			return "pdf", true
		}
		// PNG signature: 89 50 4E 47
		if sig[0] == 0x89 && sig[1] == 0x50 && sig[2] == 0x4E && sig[3] == 0x47 {
			return "image", true
		}
		// GIF signature: 47 49 46 38
		if sig[0] == 0x47 && sig[1] == 0x49 && sig[2] == 0x46 && sig[3] == 0x38 {
			return "image", true
		}
	}
	if len(sig) >= 3 {
		// JPEG signature: FF D8 FF
		if sig[0] == 0xFF && sig[1] == 0xD8 && sig[2] == 0xFF {
			return "image", true
		}
	}
	// WebP signature: RIFF....WEBP
	if len(sig) >= 12 && sig[0] == 0x52 && sig[1] == 0x49 && sig[2] == 0x46 && sig[3] == 0x46 &&
		sig[8] == 0x57 && sig[9] == 0x45 && sig[10] == 0x42 && sig[11] == 0x50 {
		return "image", true
	}
	return "", false
}
