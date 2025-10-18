package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"goodpack-server/config"
	"goodpack-server/models"
	"goodpack-server/repository"
)

type ProductHandler struct {
	repo         *repository.ProductRepository
	configLoader *config.ConfigLoader
}

func NewProductHandler(repo *repository.ProductRepository) *ProductHandler {
	configLoader := config.NewConfigLoader()
	if err := configLoader.LoadConfig(); err != nil {
		// If config loading fails, continue with empty config
		// Log error but don't fail the handler creation
	}

	return &ProductHandler{
		repo:         repo,
		configLoader: configLoader,
	}
}

func (h *ProductHandler) GetProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	products, err := h.repo.GetAll(r.Context())
	if err != nil {
		http.Error(w, "Failed to get products", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(products)
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	// Try to get by ObjectID first, then by SKU ID
	product, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		// If not found by ObjectID, try SKU ID
		product, err = h.repo.GetBySKUID(r.Context(), id)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var productReq models.ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&productReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	product := productReq.ToProduct()
	if err := h.repo.Create(r.Context(), product); err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	// Get existing product - try by ObjectID first, then by SKUID
	existingProduct, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		// Try to find by SKUID if ObjectID fails
		existingProduct, err = h.repo.GetBySKUID(r.Context(), id)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	var productReq models.ProductRequest
	if err := json.NewDecoder(r.Body).Decode(&productReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update existing product
	existingProduct.UpdateFromRequest(&productReq)
	if err := h.repo.Update(r.Context(), existingProduct.ID.Hex(), existingProduct); err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(existingProduct)
}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	var stockReq models.StockUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&stockReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateStock(r.Context(), id, stockReq.Stock); err != nil {
		http.Error(w, "Failed to update stock", http.StatusInternalServerError)
		return
	}

	// Get updated product
	product, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	products, err := h.repo.GetAll(r.Context())
	if err != nil {
		http.Error(w, "Failed to get inventory", http.StatusInternalServerError)
		return
	}

	// Calculate inventory summary
	var totalProducts, totalStock, lowStockProducts, outOfStockProducts int
	for _, product := range products {
		totalProducts++
		totalStock += product.GetTotalStock()
		if product.GetTotalStock() == 0 {
			outOfStockProducts++
		} else if product.IsLowStock() {
			lowStockProducts++
		}
	}

	inventory := map[string]interface{}{
		"products":           products,
		"totalProducts":      totalProducts,
		"totalStock":         totalStock,
		"lowStockProducts":   lowStockProducts,
		"outOfStockProducts": outOfStockProducts,
	}

	json.NewEncoder(w).Encode(inventory)
}

func (h *ProductHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	categories, err := h.repo.GetCategories(r.Context())
	if err != nil {
		http.Error(w, "Failed to get categories", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(categories)
}

func (h *ProductHandler) UpdatePrice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	var priceReq models.PriceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&priceReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdatePrice(r.Context(), id, priceReq.Price); err != nil {
		http.Error(w, "Failed to update price", http.StatusInternalServerError)
		return
	}

	// Get updated product
	product, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) GetByCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	category := vars["category"]

	products, err := h.repo.GetByCategory(r.Context(), category)
	if err != nil {
		http.Error(w, "Failed to get products by category", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(products)
}

func (h *ProductHandler) GetLowStockProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get threshold from query parameter (default: 10)
	threshold := 10
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		// Parse threshold parameter if provided
		// For now, we'll use default value
	}

	products, err := h.repo.GetLowStockProducts(r.Context(), threshold)
	if err != nil {
		http.Error(w, "Failed to get low stock products", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(products)
}

// GetConfigCategories returns all categories from config
func (h *ProductHandler) GetConfigCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	categories := h.configLoader.GetCategories()
	json.NewEncoder(w).Encode(categories)
}

// GetConfigColors returns all colors from config
func (h *ProductHandler) GetConfigColors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	colors := h.configLoader.GetColors()
	json.NewEncoder(w).Encode(colors)
}

// GetConfigAccounts returns all active accounts from config
func (h *ProductHandler) GetConfigAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	accounts := h.configLoader.GetActiveAccounts()
	json.NewEncoder(w).Encode(accounts)
}

// UploadProductImage handles product image upload
func (h *ProductHandler) UploadProductImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productId := vars["id"]

	// Parse multipart form with 10MB max memory
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get the file from form data
	file, handler, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "No image file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file size (max 5MB)
	if handler.Size > 5*1024*1024 {
		http.Error(w, "File size too large. Maximum size is 5MB", http.StatusBadRequest)
		return
	}

	// Check file type by reading file signature
	fileBytes := make([]byte, 12)
	_, err = file.Read(fileBytes)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}

	// Reset file position
	file.Seek(0, 0)

	// Check file signature
	isValidType := false
	if len(fileBytes) >= 3 {
		// JPEG signature: FF D8 FF
		if fileBytes[0] == 0xFF && fileBytes[1] == 0xD8 && fileBytes[2] == 0xFF {
			isValidType = true
		}
		// PNG signature: 89 50 4E 47
		if fileBytes[0] == 0x89 && fileBytes[1] == 0x50 && fileBytes[2] == 0x4E && fileBytes[3] == 0x47 {
			isValidType = true
		}
		// GIF signature: 47 49 46 38
		if fileBytes[0] == 0x47 && fileBytes[1] == 0x49 && fileBytes[2] == 0x46 && fileBytes[3] == 0x38 {
			isValidType = true
		}
		// WebP signature: 52 49 46 46 (RIFF)
		if len(fileBytes) >= 12 && fileBytes[0] == 0x52 && fileBytes[1] == 0x49 && fileBytes[2] == 0x46 && fileBytes[3] == 0x46 {
			// Check for WEBP in bytes 8-11
			if fileBytes[8] == 0x57 && fileBytes[9] == 0x45 && fileBytes[10] == 0x42 && fileBytes[11] == 0x50 {
				isValidType = true
			}
		}
	}

	if !isValidType {
		http.Error(w, "Invalid file type. Only JPEG, PNG, GIF, and WebP are allowed", http.StatusBadRequest)
		return
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "uploads/products"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	ext := filepath.Ext(handler.Filename)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%d%s", productId, timestamp, ext)
	filePath := filepath.Join(uploadDir, filename)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Get existing product first
	product, err := h.repo.GetByID(r.Context(), productId)
	if err != nil {
		// Try to find by SKUID if ObjectID fails
		product, err = h.repo.GetBySKUID(r.Context(), productId)
		if err != nil {
			// Clean up uploaded file
			os.Remove(filePath)
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	// Delete old image file if exists
	if product.ImageURL != nil && *product.ImageURL != "" {
		oldImagePath := *product.ImageURL
		// Remove /uploads/ prefix if present
		if strings.HasPrefix(oldImagePath, "/uploads/") {
			oldImagePath = strings.TrimPrefix(oldImagePath, "/uploads/")
		}

		oldFilePath := filepath.Join("uploads", oldImagePath)
		if err := os.Remove(oldFilePath); err != nil {
			// Log warning but don't fail the upload
			fmt.Printf("Warning: Failed to delete old image file %s: %v\n", oldFilePath, err)
		}
	}

	// Update product with new image URL
	imageURL := fmt.Sprintf("/uploads/products/%s", filename)
	product.ImageURL = &imageURL
	if err := h.repo.Update(r.Context(), product.ID.Hex(), product); err != nil {
		// Clean up uploaded file
		os.Remove(filePath)
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":  true,
		"message":  "Image uploaded successfully",
		"imageUrl": imageURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ServeProductImage serves product images
func (h *ProductHandler) ServeProductImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Security check - prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("uploads/products", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Set appropriate content type
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// DeleteProductImage deletes a product image
func (h *ProductHandler) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	productId := vars["id"]

	// Get existing product
	product, err := h.repo.GetByID(r.Context(), productId)
	if err != nil {
		// Try to find by SKUID if ObjectID fails
		product, err = h.repo.GetBySKUID(r.Context(), productId)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	// Check if product has an image
	if product.ImageURL == nil || *product.ImageURL == "" {
		http.Error(w, "Product has no image to delete", http.StatusBadRequest)
		return
	}

	// Delete the physical file
	imagePath := *product.ImageURL
	if imagePath != "" {
		// Remove /uploads/ prefix if present
		if strings.HasPrefix(imagePath, "/uploads/") {
			imagePath = strings.TrimPrefix(imagePath, "/uploads/")
		}

		filePath := filepath.Join("uploads", imagePath)
		if err := os.Remove(filePath); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to delete image file %s: %v\n", filePath, err)
		}
	}

	// Update product to remove image URL
	product.ImageURL = nil
	if err := h.repo.Update(r.Context(), product.ID.Hex(), product); err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":  true,
		"message":  "Image deleted successfully",
		"imageUrl": nil,
	}

	json.NewEncoder(w).Encode(response)
}
