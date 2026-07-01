// Command backfill-product-status sets the initial active/inactive status on
// existing products that predate the status field.
//
// Why: the status field was added after products already existed in the
// database, so those documents have no status set. This one-time backfill
// derives a reasonable initial value from current stock levels: a product is
// considered "active" if it still has real/physical stock (ActualStock) or
// still has VAT-tracked stock remaining, and "inactive" otherwise. Status can
// be changed manually afterwards from the UI, so this script only fills in
// products that don't have a status yet (unless --force is passed).
//
// Usage:
//
//	go run ./cmd/backfill-product-status                   # dry-run, all products (no writes)
//	go run ./cmd/backfill-product-status --apply           # WRITE computed status to the database
//	go run ./cmd/backfill-product-status --force --apply   # also overwrite products that already have a status set
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"goodpack-server/config"
	"goodpack-server/database"
	"goodpack-server/models"
	"goodpack-server/repository"
)

func main() {
	apply := flag.Bool("apply", false, "Write computed status to the database. When false (default) runs as a dry-run.")
	force := flag.Bool("force", false, "Recompute and overwrite status even for products that already have a non-empty status set.")
	flag.Parse()

	cfg := config.Load()

	mongoDB, err := database.NewMongoDB(cfg.MongoURI, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Close()

	productRepo := repository.NewProductRepository(mongoDB.GetCollection("products"))

	ctx := context.Background()

	if *apply {
		log.Println("MODE: APPLY — changes WILL be written to the database.")
	} else {
		log.Println("MODE: DRY-RUN — no changes will be written. Use --apply to persist.")
	}
	if *force {
		log.Println("--force set: products that already have a status will be recomputed too.")
	}

	products, err := productRepo.GetAll(ctx)
	if err != nil {
		log.Fatalf("Failed to load products: %v", err)
	}
	log.Printf("Loaded %d products", len(products))

	var skipped, changed, unchanged int

	for _, p := range products {
		computed := models.ProductStatusInactive
		if p.Stock.ActualStock > 0 || p.Stock.VAT.Remaining > 0 {
			computed = models.ProductStatusActive
		}

		if p.Status != "" && !*force {
			skipped++
			fmt.Printf("[%s] %s | current=%q computed=%q | SKIP (already set)\n", p.SKUID, p.Name, p.Status, computed)
			continue
		}

		if computed == p.Status {
			unchanged++
			fmt.Printf("[%s] %s | current=%q computed=%q | unchanged\n", p.SKUID, p.Name, p.Status, computed)
			continue
		}

		changed++
		action := "would change"
		if *apply {
			if err := productRepo.UpdateStatus(ctx, p.ID.Hex(), computed); err != nil {
				log.Printf("ERROR updating product %s (%s): %v", p.SKUID, p.ID.Hex(), err)
				continue
			}
			action = "changed"
		}
		fmt.Printf("[%s] %s | current=%q -> computed=%q | %s\n", p.SKUID, p.Name, p.Status, computed, action)
	}

	log.Printf("Summary: %d total, %d skipped (already set), %d unchanged, %d changed/would-change", len(products), skipped, unchanged, changed)

	if !*apply {
		log.Println("Dry-run complete. No changes written. Re-run with --apply to persist.")
	} else {
		log.Println("APPLY complete.")
	}
}
