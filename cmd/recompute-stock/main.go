// Command recompute-stock rebuilds every product's running stock totals and
// weighted-average prices from the source-of-truth purchase/sale collections.
//
// Why: a previous bug double-counted stock when a purchase/sale was edited.
// The bug is fixed, but the running totals (purchased/sold/remaining/actualStock)
// and the weighted-average prices that were written to the products collection
// are still wrong. Because those values are stored (not computed on read), they
// have to be rebuilt. Opening balances and manual adjustments (initialStock,
// actualStockInitial) are never touched by purchase/sale logic, so they are
// preserved and used as the baseline.
//
// The rebuild reuses the exact same helpers the live handlers use
// (handlers.ApplyStockAdjustment + Product.UpdatePrice) so the result matches
// what the system would have produced had the bug never existed.
//
// Usage:
//
//	go run ./cmd/recompute-stock                 # dry-run, all products (no writes)
//	go run ./cmd/recompute-stock --sku XY-0001   # dry-run, single product
//	go run ./cmd/recompute-stock --report out.csv # dry-run + CSV diff report
//	go run ./cmd/recompute-stock --apply          # WRITE changes to the database
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"goodpack-server/config"
	"goodpack-server/database"
	"goodpack-server/handlers"
	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

func main() {
	apply := flag.Bool("apply", false, "Write recomputed values to the database. When false (default) runs as a dry-run.")
	sku := flag.String("sku", "", "Only recompute the product with this SKUID (for testing/verification).")
	reportPath := flag.String("report", "", "Optional path to write a CSV diff report of all changed products.")
	flag.Parse()

	cfg := config.Load()

	mongoDB, err := database.NewMongoDB(cfg.MongoURI, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Close()

	productRepo := repository.NewProductRepository(mongoDB.GetCollection("products"))
	purchaseRepo := repository.NewPurchaseRepository(mongoDB.GetCollection("purchases"))
	saleRepo := repository.NewSaleRepository(mongoDB.GetCollection("sales"))

	ctx := context.Background()

	if *apply {
		log.Println("MODE: APPLY — changes WILL be written to the database.")
	} else {
		log.Println("MODE: DRY-RUN — no changes will be written. Use --apply to persist.")
	}

	// ---- Load everything into memory ----
	products, err := productRepo.GetAll(ctx)
	if err != nil {
		log.Fatalf("Failed to load products: %v", err)
	}
	purchases, err := purchaseRepo.GetAll(ctx)
	if err != nil {
		log.Fatalf("Failed to load purchases: %v", err)
	}
	sales, err := saleRepo.GetAll(ctx)
	if err != nil {
		log.Fatalf("Failed to load sales: %v", err)
	}
	log.Printf("Loaded %d products, %d purchases, %d sales", len(products), len(purchases), len(sales))

	// Index products by their ObjectID hex string (the id used in transaction items).
	productByID := make(map[string]*models.Product, len(products))
	for _, p := range products {
		productByID[p.ID.Hex()] = p
		// Snapshot the "before" state so we can report diffs after the rebuild.
		snapshotBefore(p)
	}

	// ---- Step 1: reset running totals to baseline (preserve opening balances) ----
	for _, p := range products {
		resetToBaseline(p)
	}

	// ---- Step 2: build a single chronologically-ordered event stream ----
	events := buildEvents(purchases, sales)

	// ---- Steps 3 & 4: replay every purchase/sale exactly like the handlers ----
	missing := map[string]int{} // productID -> count of skipped references
	for _, ev := range events {
		applyEvent(ev, productByID, missing)
	}
	for id, n := range missing {
		log.Printf("WARNING: %d transaction item(s) reference missing product %s (skipped)", n, id)
	}

	// ---- Step 5: recompute YTD/MTD strictly from transaction dates ----
	// UpdatePrice keys YTD/MTD off the wall clock at call time, which is wrong
	// when replaying historical data. Override those buckets from the events.
	recomputeYTDMTD(events, productByID)

	// ---- Report diffs ----
	var changed []*models.Product
	for _, p := range products {
		if *sku != "" && p.SKUID != *sku {
			continue
		}
		if productChanged(p) {
			changed = append(changed, p)
		}
	}

	printDiffs(changed)
	log.Printf("Products with differences: %d / %d", len(changed), len(products))

	if *reportPath != "" {
		if err := writeReport(*reportPath, products, *sku); err != nil {
			log.Printf("WARNING: failed to write report: %v", err)
		} else {
			log.Printf("Wrote CSV diff report to %s", *reportPath)
		}
	}

	// ---- Step 6: persist (only with --apply) ----
	if !*apply {
		log.Println("Dry-run complete. No changes written.")
		return
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	updated := 0
	for _, p := range products {
		if *sku != "" && p.SKUID != *sku {
			continue
		}
		if !productChanged(p) {
			continue
		}
		p.UpdatedAt = utils.NowInThailand()
		if err := productRepo.Update(writeCtx, p.ID.Hex(), p); err != nil {
			log.Printf("ERROR updating product %s (%s): %v", p.SKUID, p.ID.Hex(), err)
			continue
		}
		updated++
	}
	log.Printf("APPLY complete. Updated %d product(s).", updated)
}

// stockEvent is one purchase or sale, normalized for chronological replay.
type stockEvent struct {
	when       time.Time
	isPurchase bool
	isVAT      bool
	items      []eventItem
}

type eventItem struct {
	productID        string
	quantity         int
	unitPrice        float64
	preformProductID string  // "" if none (purchase only)
	preformUnitPrice float64 // added to effective purchase price
}

// buildEvents flattens purchases and sales into one slice sorted oldest-first.
// Sorting oldest-first makes "latest" price end up correct after replay.
func buildEvents(purchases []*models.Purchase, sales []models.Sale) []stockEvent {
	events := make([]stockEvent, 0, len(purchases)+len(sales))

	for _, pur := range purchases {
		items := make([]eventItem, 0, len(pur.Items))
		for _, it := range pur.Items {
			ei := eventItem{
				productID: it.ProductID,
				quantity:  it.Quantity,
				unitPrice: it.UnitPrice,
			}
			if it.PreformProductID != nil {
				ei.preformProductID = *it.PreformProductID
			}
			if it.PreformUnitPrice != nil {
				ei.preformUnitPrice = *it.PreformUnitPrice
			}
			items = append(items, ei)
		}
		events = append(events, stockEvent{
			when:       eventTime(pur.PurchaseDate, pur.CreatedAt),
			isPurchase: true,
			isVAT:      pur.IsVAT,
			items:      items,
		})
	}

	for _, sale := range sales {
		items := make([]eventItem, 0, len(sale.Items))
		for _, it := range sale.Items {
			items = append(items, eventItem{
				productID: it.ProductID,
				quantity:  it.Quantity,
				unitPrice: it.UnitPrice,
			})
		}
		events = append(events, stockEvent{
			when:       eventTime(sale.SaleDate, sale.CreatedAt),
			isPurchase: false,
			isVAT:      sale.IsVAT,
			items:      items,
		})
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].when.Before(events[j].when)
	})
	return events
}

// applyEvent replays a single event onto the in-memory products, mirroring the
// purchase/sale handlers exactly (including preform stock deduction).
func applyEvent(ev stockEvent, productByID map[string]*models.Product, missing map[string]int) {
	stockType := models.StockTypeNonVAT
	if ev.isVAT {
		stockType = models.StockTypeVAT
	}

	for _, it := range ev.items {
		product, ok := productByID[it.productID]
		if !ok {
			missing[it.productID]++
			continue
		}

		if ev.isPurchase {
			effectivePrice := it.unitPrice + it.preformUnitPrice
			product.UpdatePrice(effectivePrice, ev.isVAT, true, it.quantity)
			handlers.ApplyStockAdjustment(product, models.AdjustmentTypeAdd, stockType, it.quantity)

			if it.preformProductID != "" {
				if preform, ok := productByID[it.preformProductID]; ok {
					handlers.ApplyStockAdjustment(preform, models.AdjustmentTypeReduce, stockType, it.quantity)
				} else {
					missing[it.preformProductID]++
				}
			}
		} else {
			product.UpdatePrice(it.unitPrice, ev.isVAT, false, it.quantity)
			handlers.ApplyStockAdjustment(product, models.AdjustmentTypeReduce, stockType, it.quantity)
		}
	}
}

// resetToBaseline zeroes the running totals back to the pre-transaction state.
// initialStock / actualStockInitial are preserved (only manual adjustments set
// them; the bug never touched them).
func resetToBaseline(p *models.Product) {
	p.Stock.VAT.Purchased = 0
	p.Stock.VAT.Sold = 0
	p.Stock.VAT.Remaining = p.Stock.VAT.InitialStock

	p.Stock.NonVAT.Purchased = 0
	p.Stock.NonVAT.Sold = 0
	p.Stock.NonVAT.Remaining = p.Stock.NonVAT.InitialStock

	p.Stock.ActualStock = p.Stock.ActualStockInitial

	resetPriceInfo(&p.Price.PurchaseVAT)
	resetPriceInfo(&p.Price.PurchaseNonVAT)
	resetPriceInfo(&p.Price.SaleVAT)
	resetPriceInfo(&p.Price.SaleNonVAT)
	// p.Price.SalesTiers is intentionally preserved (manually configured).
}

func resetPriceInfo(pi *models.PriceInfo) {
	*pi = models.PriceInfo{}
}

// recomputeYTDMTD rebuilds the YTD/MTD price buckets from transaction dates so
// they reflect the current year/month rather than the replay wall-clock.
func recomputeYTDMTD(events []stockEvent, productByID map[string]*models.Product) {
	now := utils.NowInThailand()
	year := now.Year()
	month := int(now.Month())

	// Accumulators keyed by product id, then by price bucket.
	type acc struct {
		ytdQty, mtdQty       int
		ytdAmount, mtdAmount float64
		ytdSeen, mtdSeen     bool
	}
	// bucket index: 0=purchaseVAT 1=purchaseNonVAT 2=saleVAT 3=saleNonVAT
	accs := map[string]*[4]acc{}

	bucketIndex := func(isPurchase, isVAT bool) int {
		switch {
		case isPurchase && isVAT:
			return 0
		case isPurchase && !isVAT:
			return 1
		case !isPurchase && isVAT:
			return 2
		default:
			return 3
		}
	}

	for _, ev := range events {
		inYear := ev.when.Year() == year
		inMonth := inYear && int(ev.when.Month()) == month
		if !inYear {
			continue
		}
		idx := bucketIndex(ev.isPurchase, ev.isVAT)
		for _, it := range ev.items {
			if _, ok := productByID[it.productID]; !ok {
				continue
			}
			price := it.unitPrice
			if ev.isPurchase {
				price += it.preformUnitPrice
			}
			amount := price * float64(it.quantity)

			a := accs[it.productID]
			if a == nil {
				a = &[4]acc{}
				accs[it.productID] = a
			}
			a[idx].ytdSeen = true
			a[idx].ytdQty += it.quantity
			a[idx].ytdAmount += amount
			if inMonth {
				a[idx].mtdSeen = true
				a[idx].mtdQty += it.quantity
				a[idx].mtdAmount += amount
			}
		}
	}

	for id, a := range accs {
		p := productByID[id]
		buckets := [4]*models.PriceInfo{
			&p.Price.PurchaseVAT, &p.Price.PurchaseNonVAT,
			&p.Price.SaleVAT, &p.Price.SaleNonVAT,
		}
		for i, pi := range buckets {
			applyPeriodAcc(pi, year, month, a[i])
		}
	}
}

func applyPeriodAcc(pi *models.PriceInfo, year, month int, a struct {
	ytdQty, mtdQty       int
	ytdAmount, mtdAmount float64
	ytdSeen, mtdSeen     bool
}) {
	if a.ytdSeen {
		pi.YTDYear = year
		pi.YTDQuantity = a.ytdQty
		pi.YTDTotalAmount = a.ytdAmount
		if a.ytdQty > 0 {
			pi.AverageYTD = round2(a.ytdAmount / float64(a.ytdQty))
		}
	}
	if a.mtdSeen {
		pi.MTDYear = year
		pi.MTDMonth = month
		pi.MTDQuantity = a.mtdQty
		pi.MTDTotalAmount = a.mtdAmount
		if a.mtdQty > 0 {
			pi.AverageMTD = round2(a.mtdAmount / float64(a.mtdQty))
		}
	}
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func eventTime(primary, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary
	}
	return fallback
}

// --- before/after snapshot & diff reporting ---

var beforeSnapshot = map[string]models.Stock{}
var beforePrice = map[string]models.Price{}

func snapshotBefore(p *models.Product) {
	beforeSnapshot[p.ID.Hex()] = p.Stock
	beforePrice[p.ID.Hex()] = p.Price
}

func productChanged(p *models.Product) bool {
	id := p.ID.Hex()
	return beforeSnapshot[id] != p.Stock ||
		priceChanged(beforePrice[id], p.Price)
}

// priceChanged compares the price buckets that the recompute touches.
// SalesTiers are preserved, so they are excluded from the comparison.
func priceChanged(a, b models.Price) bool {
	return a.PurchaseVAT != b.PurchaseVAT ||
		a.PurchaseNonVAT != b.PurchaseNonVAT ||
		a.SaleVAT != b.SaleVAT ||
		a.SaleNonVAT != b.SaleNonVAT
}

func printDiffs(changed []*models.Product) {
	for _, p := range changed {
		before := beforeSnapshot[p.ID.Hex()]
		fmt.Printf("\n[%s] %s\n", p.SKUID, p.Name)
		printStockLine("  VAT    ", before.VAT, p.Stock.VAT)
		printStockLine("  NonVAT ", before.NonVAT, p.Stock.NonVAT)
		fmt.Printf("  ActualStock: %d -> %d\n", before.ActualStock, p.Stock.ActualStock)
	}
}

func printStockLine(label string, b, a models.StockInfo) {
	if b == a {
		return
	}
	fmt.Printf("%s purchased %d->%d  sold %d->%d  remaining %d->%d\n",
		label, b.Purchased, a.Purchased, b.Sold, a.Sold, b.Remaining, a.Remaining)
}

func writeReport(path string, products []*models.Product, sku string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"skuId", "name",
		"vatPurchased_before", "vatPurchased_after",
		"vatSold_before", "vatSold_after",
		"vatRemaining_before", "vatRemaining_after",
		"nonVatPurchased_before", "nonVatPurchased_after",
		"nonVatSold_before", "nonVatSold_after",
		"nonVatRemaining_before", "nonVatRemaining_after",
		"actualStock_before", "actualStock_after",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, p := range products {
		if sku != "" && p.SKUID != sku {
			continue
		}
		if !productChanged(p) {
			continue
		}
		b := beforeSnapshot[p.ID.Hex()]
		a := p.Stock
		row := []string{
			p.SKUID, p.Name,
			itoa(b.VAT.Purchased), itoa(a.VAT.Purchased),
			itoa(b.VAT.Sold), itoa(a.VAT.Sold),
			itoa(b.VAT.Remaining), itoa(a.VAT.Remaining),
			itoa(b.NonVAT.Purchased), itoa(a.NonVAT.Purchased),
			itoa(b.NonVAT.Sold), itoa(a.NonVAT.Sold),
			itoa(b.NonVAT.Remaining), itoa(a.NonVAT.Remaining),
			itoa(b.ActualStock), itoa(a.ActualStock),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func itoa(i int) string { return strconv.Itoa(i) }
