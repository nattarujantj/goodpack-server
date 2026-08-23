package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"goodpack-server/config"
	"goodpack-server/database"
	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/routes"
	"goodpack-server/services"
	"goodpack-server/utils"
)

func main() {
	cfg := config.Load()

	mongoDB, err := database.NewMongoDB(cfg.MongoURI, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Close()

	productRepo := repository.NewProductRepository(mongoDB.GetCollection("products"))
	customerRepo := repository.NewCustomerRepository(mongoDB.GetCollection("customers"))
	supplierRepo := repository.NewSupplierRepository(mongoDB.GetCollection("suppliers"))
	purchaseRepo := repository.NewPurchaseRepository(mongoDB.GetCollection("purchases"))
	saleRepo := repository.NewSaleRepository(mongoDB.GetCollection("sales"))
	quotationRepo := repository.NewQuotationRepository(mongoDB.GetCollection("quotations"))
	purchaseOrderRepo := repository.NewPurchaseOrderRepository(mongoDB.GetCollection("purchase_orders"))
	stockAdjustmentRepo := repository.NewStockAdjustmentRepository(mongoDB.GetCollection("stock_adjustments"))
	shippingCompanyRepo := repository.NewShippingCompanyRepository(mongoDB.GetCollection("shipping_companies"))
	internationalImportRepo := repository.NewInternationalImportRepository(mongoDB.GetCollection("international_imports"))
	fclShipmentRepo := repository.NewFCLShipmentRepository(mongoDB.GetCollection("fcl_shipments"))
	expenseRepo := repository.NewExpenseRepository(mongoDB.GetCollection("expenses"))
	userRepo := repository.NewUserRepository(mongoDB.GetCollection("users"))
	inventorySnapshotRepo := repository.NewInventorySnapshotRepository(mongoDB.GetCollection("inventory_snapshots"))

	autoSeedSuperAdmin(cfg, userRepo)

	jwtExpiry, err := time.ParseDuration(cfg.JWTExpiry)
	if err != nil {
		log.Printf("Invalid JWT_EXPIRY '%s', using default 24h", cfg.JWTExpiry)
		jwtExpiry = 24 * time.Hour
	}

	// Start monthly auto-snapshot scheduler
	snapshotSvc := services.NewInventorySnapshotService(inventorySnapshotRepo, productRepo)
	services.StartSnapshotScheduler(snapshotSvc)

	router := routes.SetupRoutes(
		productRepo, customerRepo, supplierRepo,
		purchaseRepo, saleRepo, quotationRepo,
		purchaseOrderRepo, stockAdjustmentRepo,
		shippingCompanyRepo, internationalImportRepo,
		fclShipmentRepo,
		expenseRepo, userRepo, inventorySnapshotRepo,
		cfg.JWTSecret, jwtExpiry,
	)

	log.Printf("Server starting on port :%s", cfg.Port)
	log.Printf("API Base URL: http://localhost:%s/api", cfg.Port)
	log.Printf("Health Check: http://localhost:%s/api/health", cfg.Port)
	log.Printf("Database: MongoDB (%s)", cfg.Database)

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func autoSeedSuperAdmin(cfg *config.Config, userRepo *repository.UserRepository) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := userRepo.CountUsers(ctx)
	if err != nil {
		log.Printf("Warning: could not check user count: %v", err)
		return
	}

	if count > 0 {
		return
	}

	if cfg.AdminPassword == "" {
		log.Println("No users in database and ADMIN_PASSWORD not set. Skipping auto-seed.")
		log.Println("Set ADMIN_PASSWORD in .env to auto-create a super admin on startup.")
		return
	}

	now := utils.NowInThailand()
	user := &models.User{
		Username:    cfg.AdminUsername,
		DisplayName: cfg.AdminDisplayName,
		Role:        models.RoleSuperAdmin,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := user.SetPassword(cfg.AdminPassword); err != nil {
		log.Printf("Failed to hash admin password: %v", err)
		return
	}

	if err := userRepo.Create(ctx, user); err != nil {
		log.Printf("Failed to create super admin: %v", err)
		return
	}

	log.Printf("Auto-created super admin user: %s", cfg.AdminUsername)
}
