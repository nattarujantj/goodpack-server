package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goodpack-server/models"
)

type InventorySnapshotRepository struct {
	collection *mongo.Collection
}

func NewInventorySnapshotRepository(collection *mongo.Collection) *InventorySnapshotRepository {
	return &InventorySnapshotRepository{collection: collection}
}

// Upsert inserts or replaces a snapshot identified by {month, year}
func (r *InventorySnapshotRepository) Upsert(ctx context.Context, snapshot *models.InventorySnapshot) error {
	filter := bson.M{"month": snapshot.Month, "year": snapshot.Year}
	opts := options.Replace().SetUpsert(true)
	_, err := r.collection.ReplaceOne(ctx, filter, snapshot, opts)
	return err
}

// FindByMonthYear returns the snapshot for the given month/year, or nil if not found
func (r *InventorySnapshotRepository) FindByMonthYear(ctx context.Context, month, year int) (*models.InventorySnapshot, error) {
	var snap models.InventorySnapshot
	err := r.collection.FindOne(ctx, bson.M{"month": month, "year": year}).Decode(&snap)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// FindAll returns all snapshots sorted by year desc, month desc
func (r *InventorySnapshotRepository) FindAll(ctx context.Context) ([]*models.InventorySnapshot, error) {
	opts := options.Find().SetSort(bson.D{
		{Key: "year", Value: -1},
		{Key: "month", Value: -1},
	})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var snapshots []*models.InventorySnapshot
	if err := cursor.All(ctx, &snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}
