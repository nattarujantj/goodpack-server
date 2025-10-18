package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"goodpack-server/handlers"
	"goodpack-server/repository"
)

func SetupRoutes(productRepo *repository.ProductRepository, customerRepo *repository.CustomerRepository, purchaseRepo *repository.PurchaseRepository, saleRepo *repository.SaleRepository, quotationRepo *repository.QuotationRepository) http.Handler {
	router := mux.NewRouter()

	// Initialize handlers test2
	productHandler := handlers.NewProductHandler(productRepo)
	customerHandler := handlers.NewCustomerHandler(customerRepo)
	purchaseHandler := handlers.NewPurchaseHandler(purchaseRepo, customerRepo, productRepo)
	saleHandler := handlers.NewSaleHandler(saleRepo, customerRepo, productRepo, quotationRepo)
	quotationHandler := handlers.NewQuotationHandler(quotationRepo, customerRepo, productRepo)
	migrationHandler := handlers.NewMigrationHandler(customerRepo, productRepo, purchaseRepo, saleRepo)

	// API routes
	api := router.PathPrefix("/api").Subrouter()

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

	// Categories routes
	api.HandleFunc("/categories", productHandler.GetCategories).Methods("GET")
	api.HandleFunc("/config/categories", productHandler.GetConfigCategories).Methods("GET")
	api.HandleFunc("/config/colors", productHandler.GetConfigColors).Methods("GET")
	api.HandleFunc("/config/accounts", productHandler.GetConfigAccounts).Methods("GET")

	// Customer routes
	api.HandleFunc("/customers", customerHandler.GetCustomers).Methods("GET")
	api.HandleFunc("/customers", customerHandler.CreateCustomer).Methods("POST")
	api.HandleFunc("/customers/{id}", customerHandler.GetCustomer).Methods("GET")
	api.HandleFunc("/customers/{id}", customerHandler.UpdateCustomer).Methods("PUT")
	api.HandleFunc("/customers/{id}", customerHandler.DeleteCustomer).Methods("DELETE")

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

	// Migration routes
	api.HandleFunc("/migration/customers/csv", migrationHandler.MigrateCustomersFromCSV).Methods("POST")
	api.HandleFunc("/migration/customers/template", migrationHandler.GetCustomerCSVTemplate).Methods("GET")
	api.HandleFunc("/migration/products/csv", migrationHandler.MigrateProductsFromCSV).Methods("POST")
	api.HandleFunc("/migration/products/template", migrationHandler.GetProductCSVTemplate).Methods("GET")
	api.HandleFunc("/migration/purchases/csv", migrationHandler.MigratePurchasesFromCSV).Methods("POST")
	api.HandleFunc("/migration/purchases/template", migrationHandler.GetPurchaseCSVTemplate).Methods("GET")
	api.HandleFunc("/migration/sales/csv", migrationHandler.MigrateSalesFromCSV).Methods("POST")
	api.HandleFunc("/migration/sales/template", migrationHandler.GetSaleCSVTemplate).Methods("GET")
	api.HandleFunc("/migration/status", migrationHandler.GetMigrationStatus).Methods("GET")

	// Static file serving for uploaded images
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads/"))))

	// Health check
	api.HandleFunc("/health", healthCheck).Methods("GET")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)
	return handler
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"database":  "mongodb",
	}
	json.NewEncoder(w).Encode(response)
}
