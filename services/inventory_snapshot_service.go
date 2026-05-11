package services

import (
	"context"

	"goodpack-server/models"
	"goodpack-server/repository"
	"goodpack-server/utils"
)

type InventorySnapshotService struct {
	snapshotRepo *repository.InventorySnapshotRepository
	productRepo  *repository.ProductRepository
}

func NewInventorySnapshotService(
	snapshotRepo *repository.InventorySnapshotRepository,
	productRepo *repository.ProductRepository,
) *InventorySnapshotService {
	return &InventorySnapshotService{
		snapshotRepo: snapshotRepo,
		productRepo:  productRepo,
	}
}

// TakeSnapshot captures current product stock into a monthly snapshot.
// isManual=true always overwrites; isManual=false (auto) skips if snapshot already exists.
func (s *InventorySnapshotService) TakeSnapshot(ctx context.Context, month, year int, createdBy string, isManual bool) (*models.InventorySnapshot, error) {
	if !isManual {
		existing, err := s.snapshotRepo.FindByMonthYear(ctx, month, year)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	products, err := s.productRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]models.ProductSnapshotItem, 0, len(products))
	var totalVAT, totalNonVAT, totalActual int

	for _, p := range products {
		item := models.ProductSnapshotItem{
			ProductID:       p.ID.Hex(),
			SKUID:           p.SKUID,
			Code:            p.Code,
			Name:            p.Name,
			Category:        p.Category,
			Color:           p.Color,
			Size:            p.Size,
			VATRemaining:    p.Stock.VAT.Remaining,
			NonVATRemaining: p.Stock.NonVAT.Remaining,
			ActualStock:     p.Stock.ActualStock,
		}
		items = append(items, item)
		totalVAT += p.Stock.VAT.Remaining
		totalNonVAT += p.Stock.NonVAT.Remaining
		totalActual += p.Stock.ActualStock
	}

	snap := &models.InventorySnapshot{
		Month:            month,
		Year:             year,
		SnapshotDate:     utils.NowInThailand(),
		CreatedBy:        createdBy,
		IsManual:         isManual,
		Products:         items,
		TotalProducts:    len(items),
		TotalVATStock:    totalVAT,
		TotalNonVATStock: totalNonVAT,
		TotalActualStock: totalActual,
	}

	if err := s.snapshotRepo.Upsert(ctx, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

// UpdateSnapshot replaces the product list in an existing snapshot and recalculates totals.
func (s *InventorySnapshotService) UpdateSnapshot(ctx context.Context, month, year int, products []models.ProductSnapshotItem, createdBy string) (*models.InventorySnapshot, error) {
	var totalVAT, totalNonVAT, totalActual int
	for _, p := range products {
		totalVAT += p.VATRemaining
		totalNonVAT += p.NonVATRemaining
		totalActual += p.ActualStock
	}

	snap := &models.InventorySnapshot{
		Month:            month,
		Year:             year,
		SnapshotDate:     utils.NowInThailand(),
		CreatedBy:        createdBy,
		IsManual:         true,
		Products:         products,
		TotalProducts:    len(products),
		TotalVATStock:    totalVAT,
		TotalNonVATStock: totalNonVAT,
		TotalActualStock: totalActual,
	}

	if err := s.snapshotRepo.Upsert(ctx, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

// GetSnapshot returns the snapshot for a given month/year (nil if not found)
func (s *InventorySnapshotService) GetSnapshot(ctx context.Context, month, year int) (*models.InventorySnapshot, error) {
	return s.snapshotRepo.FindByMonthYear(ctx, month, year)
}

// ListSnapshots returns all snapshots sorted newest first
func (s *InventorySnapshotService) ListSnapshots(ctx context.Context) ([]*models.InventorySnapshot, error) {
	return s.snapshotRepo.FindAll(ctx)
}
