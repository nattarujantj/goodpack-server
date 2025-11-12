package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goodpack-server/models"
)

type StockAdjustmentRepository struct {
	collection *mongo.Collection
}

func NewStockAdjustmentRepository(collection *mongo.Collection) *StockAdjustmentRepository {
	return &StockAdjustmentRepository{
		collection: collection,
	}
}

// Create creates a new stock adjustment record
func (r *StockAdjustmentRepository) Create(ctx context.Context, adjustment *models.StockAdjustment) error {
	if adjustment.ID.IsZero() {
		adjustment.ID = primitive.NewObjectID()
	}
	_, err := r.collection.InsertOne(ctx, adjustment)
	return err
}

// GetByID gets a stock adjustment by ID
func (r *StockAdjustmentRepository) GetByID(ctx context.Context, id string) (*models.StockAdjustment, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var adjustment models.StockAdjustment
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&adjustment)
	if err != nil {
		return nil, err
	}

	return &adjustment, nil
}

// GetByProductID gets all stock adjustments for a specific product
func (r *StockAdjustmentRepository) GetByProductID(ctx context.Context, productID string, limit int) ([]*models.StockAdjustment, error) {
	filter := bson.M{"productId": productID}

	opts := options.Find()
	opts.SetSort(bson.M{"createdAt": -1}) // Sort by newest first
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var adjustments []*models.StockAdjustment
	for cursor.Next(ctx) {
		var adjustment models.StockAdjustment
		if err := cursor.Decode(&adjustment); err != nil {
			continue
		}
		adjustments = append(adjustments, &adjustment)
	}

	return adjustments, cursor.Err()
}

// GetByProductIDAndDateRange gets stock adjustments for a product within a date range
func (r *StockAdjustmentRepository) GetByProductIDAndDateRange(ctx context.Context, productID string, startDate, endDate time.Time, limit int) ([]*models.StockAdjustment, error) {
	filter := bson.M{
		"productId": productID,
		"createdAt": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	opts := options.Find()
	opts.SetSort(bson.M{"createdAt": -1})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var adjustments []*models.StockAdjustment
	for cursor.Next(ctx) {
		var adjustment models.StockAdjustment
		if err := cursor.Decode(&adjustment); err != nil {
			continue
		}
		adjustments = append(adjustments, &adjustment)
	}

	return adjustments, cursor.Err()
}

// GetBySource gets stock adjustments by source type and source ID
func (r *StockAdjustmentRepository) GetBySource(ctx context.Context, sourceType models.SourceType, sourceID string) ([]*models.StockAdjustment, error) {
	filter := bson.M{
		"sourceType": sourceType,
		"sourceId":   sourceID,
	}

	opts := options.Find()
	opts.SetSort(bson.M{"createdAt": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var adjustments []*models.StockAdjustment
	for cursor.Next(ctx) {
		var adjustment models.StockAdjustment
		if err := cursor.Decode(&adjustment); err != nil {
			continue
		}
		adjustments = append(adjustments, &adjustment)
	}

	return adjustments, cursor.Err()
}

// GetAll gets all stock adjustments with pagination
func (r *StockAdjustmentRepository) GetAll(ctx context.Context, limit, skip int) ([]*models.StockAdjustment, error) {
	opts := options.Find()
	opts.SetSort(bson.M{"createdAt": -1})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if skip > 0 {
		opts.SetSkip(int64(skip))
	}

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var adjustments []*models.StockAdjustment
	for cursor.Next(ctx) {
		var adjustment models.StockAdjustment
		if err := cursor.Decode(&adjustment); err != nil {
			continue
		}
		adjustments = append(adjustments, &adjustment)
	}

	return adjustments, cursor.Err()
}

// CountByProductID counts total adjustments for a product
func (r *StockAdjustmentRepository) CountByProductID(ctx context.Context, productID string) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"productId": productID})
}

// Delete deletes a stock adjustment by ID
func (r *StockAdjustmentRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
