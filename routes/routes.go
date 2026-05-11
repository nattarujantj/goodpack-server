package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"goodpack-server/handlers"
	"goodpack-server/middleware"
	"goodpack-server/repository"
	"goodpack-server/services"
	"goodpack-server/utils"
)

func SetupRoutes(
	productRepo *repository.ProductRepository,
	customerRepo *repository.CustomerRepository,
	supplierRepo *repository.SupplierRepository,
	purchaseRepo *repository.PurchaseRepository,
	saleRepo *repository.SaleRepository,
	quotationRepo *repository.QuotationRepository,
	purchaseOrderRepo *repository.PurchaseOrderRepository,
	stockAdjustmentRepo *repository.StockAdjustmentRepository,
	shippingCompanyRepo *repository.ShippingCompanyRepository,
	internationalImportRepo *repository.InternationalImportRepository,
	expenseRepo *repository.ExpenseRepository,
	userRepo *repository.UserRepository,
	inventorySnapshotRepo *repository.InventorySnapshotRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
) http.Handler {
	router := mux.NewRouter()

	// Initialize services that are shared between handlers
	snapshotService := services.NewInventorySnapshotService(inventorySnapshotRepo, productRepo)

	// Initialize handlers
	productHandler := handlers.NewProductHandler(productRepo)
	customerHandler := handlers.NewCustomerHandler(customerRepo, saleRepo)
	supplierHandler := handlers.NewSupplierHandler(supplierRepo, purchaseRepo)
	purchaseHandler := handlers.NewPurchaseHandler(purchaseRepo, supplierRepo, productRepo, stockAdjustmentRepo)
	saleHandler := handlers.NewSaleHandler(saleRepo, customerRepo, productRepo, quotationRepo, stockAdjustmentRepo)
	quotationHandler := handlers.NewQuotationHandler(quotationRepo, customerRepo, productRepo)
	purchaseOrderHandler := handlers.NewPurchaseOrderHandler(purchaseOrderRepo, supplierRepo, productRepo)
	stockAdjustmentHandler := handlers.NewStockAdjustmentHandler(stockAdjustmentRepo, productRepo)
	exportHandler := handlers.NewExportHandler(purchaseRepo, saleRepo, customerRepo, expenseRepo, snapshotService)
	expenseHandler := handlers.NewExpenseHandler(expenseRepo)
	shippingCompanyHandler := handlers.NewShippingCompanyHandler(shippingCompanyRepo)
	internationalImportHandler := handlers.NewInternationalImportHandler(internationalImportRepo, supplierRepo, shippingCompanyRepo, productRepo, purchaseRepo, stockAdjustmentRepo)
	authHandler := handlers.NewAuthHandler(userRepo, jwtSecret, jwtExpiry)
	inventorySnapshotHandler := handlers.NewInventorySnapshotHandler(snapshotService)

	// ── Public routes (no auth required) ──
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/config/categories", productHandler.GetConfigCategories).Methods("GET")
	router.HandleFunc("/api/config/colors", productHandler.GetConfigColors).Methods("GET")
	router.HandleFunc("/api/config/accounts", productHandler.GetConfigAccounts).Methods("GET")

	// ── Protected routes (auth required) ──
	api := router.PathPrefix("/api").Subrouter()
	api.Use(middleware.AuthMiddleware(jwtSecret, userRepo))

	// Auth routes (require login)
	api.HandleFunc("/auth/me", authHandler.GetMe).Methods("GET")
	api.HandleFunc("/auth/change-password", authHandler.ChangePassword).Methods("POST")
	api.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	api.HandleFunc("/auth/users", authHandler.GetAllUsers).Methods("GET")
	api.HandleFunc("/auth/users", authHandler.DeleteUser).Methods("DELETE")

	// Product routes
	api.HandleFunc("/products", productHandler.GetProducts).Methods("GET")
	api.HandleFunc("/products", productHandler.CreateProduct).Methods("POST")
	api.HandleFunc("/products/{id}", productHandler.GetProduct).Methods("GET")
	api.HandleFunc("/products/{id}", productHandler.UpdateProduct).Methods("PUT")
	api.HandleFunc("/products/{id}", productHandler.DeleteProduct).Methods("DELETE")
	api.HandleFunc("/products/{id}/stock", productHandler.UpdateStock).Methods("PATCH")
	api.HandleFunc("/products/{id}/price", productHandler.UpdatePrice).Methods("PATCH")
	api.HandleFunc("/products/{id}/image", productHandler.UploadProductImage).Methods("POST")
	api.HandleFunc("/products/{id}/image", productHandler.DeleteProductImage).Methods("DELETE")
	api.HandleFunc("/products/category/{category}", productHandler.GetByCategory).Methods("GET")
	api.HandleFunc("/products/low-stock", productHandler.GetLowStockProducts).Methods("GET")

	// Stock Adjustment routes
	api.HandleFunc("/products/{id}/stock/adjust", stockAdjustmentHandler.AdjustStock).Methods("POST")
	api.HandleFunc("/products/{id}/stock/history", stockAdjustmentHandler.GetStockHistory).Methods("GET")
	api.HandleFunc("/stock/history", stockAdjustmentHandler.GetAllStockHistory).Methods("GET")
	api.HandleFunc("/stock/history/source", stockAdjustmentHandler.GetStockHistoryBySource).Methods("GET")
	api.HandleFunc("/stock/adjustments/{id}", stockAdjustmentHandler.DeleteStockAdjustment).Methods("DELETE")

	// Categories routes
	api.HandleFunc("/categories", productHandler.GetCategories).Methods("GET")

	// Customer routes
	api.HandleFunc("/customers", customerHandler.GetCustomers).Methods("GET")
	api.HandleFunc("/customers", customerHandler.CreateCustomer).Methods("POST")
	api.HandleFunc("/customers/{id}", customerHandler.GetCustomer).Methods("GET")
	api.HandleFunc("/customers/{id}", customerHandler.UpdateCustomer).Methods("PUT")
	api.HandleFunc("/customers/{id}", customerHandler.DeleteCustomer).Methods("DELETE")
	api.HandleFunc("/customers/{id}/sales", customerHandler.GetCustomerSales).Methods("GET")

	// Supplier routes
	api.HandleFunc("/suppliers", supplierHandler.GetSuppliers).Methods("GET")
	api.HandleFunc("/suppliers", supplierHandler.CreateSupplier).Methods("POST")
	api.HandleFunc("/suppliers/{id}", supplierHandler.GetSupplier).Methods("GET")
	api.HandleFunc("/suppliers/{id}", supplierHandler.UpdateSupplier).Methods("PUT")
	api.HandleFunc("/suppliers/{id}", supplierHandler.DeleteSupplier).Methods("DELETE")
	api.HandleFunc("/suppliers/{id}/purchases", supplierHandler.GetSupplierPurchases).Methods("GET")

	// Purchase routes
	api.HandleFunc("/purchases", purchaseHandler.GetPurchases).Methods("GET")
	api.HandleFunc("/purchases", purchaseHandler.CreatePurchase).Methods("POST")
	api.HandleFunc("/purchases/{id}", purchaseHandler.GetPurchase).Methods("GET")
	api.HandleFunc("/purchases/{id}", purchaseHandler.UpdatePurchase).Methods("PUT")
	api.HandleFunc("/purchases/{id}", purchaseHandler.DeletePurchase).Methods("DELETE")

	// Sale routes
	api.HandleFunc("/sales", saleHandler.GetSales).Methods("GET")
	api.HandleFunc("/sales", saleHandler.CreateSale).Methods("POST")
	api.HandleFunc("/sales/{id}", saleHandler.GetSale).Methods("GET")
	api.HandleFunc("/sales/{id}", saleHandler.UpdateSale).Methods("PUT")
	api.HandleFunc("/sales/{id}", saleHandler.DeleteSale).Methods("DELETE")

	// Quotation routes
	api.HandleFunc("/quotations", quotationHandler.GetAllQuotations).Methods("GET")
	api.HandleFunc("/quotations", quotationHandler.CreateQuotation).Methods("POST")
	api.HandleFunc("/quotations/{id}", quotationHandler.GetQuotation).Methods("GET")
	api.HandleFunc("/quotations/{id}", quotationHandler.UpdateQuotation).Methods("PUT")
	api.HandleFunc("/quotations/{id}", quotationHandler.DeleteQuotation).Methods("DELETE")
	api.HandleFunc("/quotations/{id}/copy-to-sale", quotationHandler.CopyToSale).Methods("GET")
	api.HandleFunc("/quotations/{id}/status", quotationHandler.UpdateQuotationStatus).Methods("PATCH")

	// Purchase Order routes
	api.HandleFunc("/purchase-orders", purchaseOrderHandler.GetAllPurchaseOrders).Methods("GET")
	api.HandleFunc("/purchase-orders", purchaseOrderHandler.CreatePurchaseOrder).Methods("POST")
	api.HandleFunc("/purchase-orders/{id}", purchaseOrderHandler.GetPurchaseOrder).Methods("GET")
	api.HandleFunc("/purchase-orders/{id}", purchaseOrderHandler.UpdatePurchaseOrder).Methods("PUT")
	api.HandleFunc("/purchase-orders/{id}", purchaseOrderHandler.DeletePurchaseOrder).Methods("DELETE")
	api.HandleFunc("/purchase-orders/{id}/copy-to-purchase", purchaseOrderHandler.CopyToPurchase).Methods("GET")
	api.HandleFunc("/purchase-orders/{id}/status", purchaseOrderHandler.UpdatePurchaseOrderStatus).Methods("PATCH")

	// Expense routes
	api.HandleFunc("/expenses", expenseHandler.GetExpenses).Methods("GET")
	api.HandleFunc("/expenses", expenseHandler.CreateExpense).Methods("POST")
	api.HandleFunc("/expenses/categories", expenseHandler.GetCategories).Methods("GET")
	api.HandleFunc("/expenses/{id}", expenseHandler.GetExpense).Methods("GET")
	api.HandleFunc("/expenses/{id}", expenseHandler.UpdateExpense).Methods("PUT")
	api.HandleFunc("/expenses/{id}", expenseHandler.DeleteExpense).Methods("DELETE")

	// Export routes
	api.HandleFunc("/export/email", exportHandler.ExportAndSendEmail).Methods("POST")

	// Inventory Snapshot routes
	api.HandleFunc("/inventory-snapshots", inventorySnapshotHandler.ListSnapshots).Methods("GET")
	api.HandleFunc("/inventory-snapshots", inventorySnapshotHandler.CreateSnapshot).Methods("POST")
	api.HandleFunc("/inventory-snapshots/{year}/{month}", inventorySnapshotHandler.GetSnapshot).Methods("GET")
	api.HandleFunc("/inventory-snapshots/{year}/{month}", inventorySnapshotHandler.UpdateSnapshot).Methods("PUT")

	// Shipping Company routes
	api.HandleFunc("/shipping-companies", shippingCompanyHandler.GetAll).Methods("GET")
	api.HandleFunc("/shipping-companies", shippingCompanyHandler.Create).Methods("POST")
	api.HandleFunc("/shipping-companies/{id}", shippingCompanyHandler.Update).Methods("PUT")
	api.HandleFunc("/shipping-companies/{id}", shippingCompanyHandler.Delete).Methods("DELETE")

	// International Import routes
	api.HandleFunc("/international-imports", internationalImportHandler.GetAll).Methods("GET")
	api.HandleFunc("/international-imports", internationalImportHandler.Create).Methods("POST")
	api.HandleFunc("/international-imports/{id}", internationalImportHandler.GetByID).Methods("GET")
	api.HandleFunc("/international-imports/{id}", internationalImportHandler.Update).Methods("PUT")
	api.HandleFunc("/international-imports/{id}", internationalImportHandler.Delete).Methods("DELETE")
	api.HandleFunc("/international-imports/{id}/create-purchase", internationalImportHandler.CreatePurchaseFromImport).Methods("POST")

	// Static file serving for uploaded images
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads/"))))

	// Static file serving for Flutter web app with SPA fallback and correct cache headers
	router.PathPrefix("/").Handler(flutterCacheMiddleware(spaHandler("web/")))

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})

	handler := c.Handler(router)
	return handler
}

// noCacheEntryPoints is the set of Flutter entry-point files that must never be cached.
var noCacheEntryPoints = map[string]bool{
	"/":                          true,
	"/index.html":                true,
	"/flutter_service_worker.js": true,
	"/flutter_bootstrap.js":      true,
	"/manifest.json":             true,
	"/main.dart.js":              true,
}

// flutterCacheMiddleware sets correct Cache-Control headers for Flutter web assets.
// Entry-point files must not be cached; hashed/versioned assets are safe to cache long-term.
func flutterCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if noCacheEntryPoints[path] {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else if isHashedAsset(path) {
			// Flutter uses content-hashed filenames for all versioned assets — safe to cache forever.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		next.ServeHTTP(w, r)
	})
}

// isHashedAsset returns true for static asset extensions produced by Flutter build.
func isHashedAsset(path string) bool {
	staticExts := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2", ".ttf", ".svg"}
	for _, ext := range staticExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// spaHandler serves Flutter web files with SPA fallback:
// if the requested file doesn't exist on disk, it serves index.html so Flutter's
// router can handle the path client-side.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": utils.NowInThailand().Format("2006-01-02 15:04:05"),
		"version":   "1.0.0",
		"database":  "mongodb",
	}
	json.NewEncoder(w).Encode(response)
}
