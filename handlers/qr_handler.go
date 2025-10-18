package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"image/png"

	"github.com/gorilla/mux"
	"github.com/skip2/go-qrcode"

	"goodpack-server/repository"
)

type QRHandler struct {
	repo *repository.ProductRepository
}

func NewQRHandler(repo *repository.ProductRepository) *QRHandler {
	return &QRHandler{
		repo: repo,
	}
}

func (h *QRHandler) GetQRCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	// Try to get by ObjectID first, then by SKU ID
	product, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		product, err = h.repo.GetBySKUID(r.Context(), id)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	// Use SKU ID for QR data
	qrData := fmt.Sprintf("https://goodpack.app/product/%s", product.SKUID)

	response := map[string]string{
		"qrCodeData":  qrData,
		"skuId":       product.SKUID,
		"productId":   product.ID.Hex(),
		"productName": product.Name,
		"productCode": product.Code,
	}

	json.NewEncoder(w).Encode(response)
}

func (h *QRHandler) GetQRCodeImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Try to get by ObjectID first, then by SKU ID
	product, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		product, err = h.repo.GetBySKUID(r.Context(), id)
		if err != nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
	}

	// Use SKU ID for QR data
	qrData := fmt.Sprintf("https://goodpack.app/product/%s", product.SKUID)

	// Generate QR code
	qr, err := qrcode.New(qrData, qrcode.Medium)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	// Convert to PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, qr.Image(256))
	if err != nil {
		http.Error(w, "Failed to encode QR code", http.StatusInternalServerError)
		return
	}

	// Set headers for image download
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"qr-%s-%s.png\"", product.SKUID, product.Name))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	// Write image data
	w.Write(buf.Bytes())
}
