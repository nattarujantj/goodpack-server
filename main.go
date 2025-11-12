package main

import (
	"log"
	"net/http"

	"goodpack-server/config"
	"goodpack-server/database"
	"goodpack-server/repository"
	"goodpack-server/routes"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to MongoDB
	mongoDB, err := database.NewMongoDB(cfg.MongoURI, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Close()

	// Initialize repositories
	productRepo := repository.NewProductRepository(mongoDB.GetCollection("products"))
	customerRepo := repository.NewCustomerRepository(mongoDB.GetCollection("customers"))
	purchaseRepo := repository.NewPurchaseRepository(mongoDB.GetCollection("purchases"))
	saleRepo := repository.NewSaleRepository(mongoDB.GetCollection("sales"))
	quotationRepo := repository.NewQuotationRepository(mongoDB.GetCollection("quotations"))
	stockAdjustmentRepo := repository.NewStockAdjustmentRepository(mongoDB.GetCollection("stock_adjustments"))

	// Setup routes
	router := routes.SetupRoutes(productRepo, customerRepo, purchaseRepo, saleRepo, quotationRepo, stockAdjustmentRepo)

	// Start server
	log.Printf("🚀 Server starting on port :%s", cfg.Port)
	log.Printf("📱 API Base URL: http://localhost:%s/api", cfg.Port)
	log.Printf("🔍 Health Check: http://localhost:%s/api/health", cfg.Port)
	log.Printf("🗄️  Database: MongoDB (%s)", cfg.Database)

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
