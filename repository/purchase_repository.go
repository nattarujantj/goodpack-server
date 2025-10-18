package repository

import (
	"context"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goodpack-server/models"
)

type PurchaseRepository struct {
	collection *mongo.Collection
}

func NewPurchaseRepository(collection *mongo.Collection) *PurchaseRepository {
	return &PurchaseRepository{
		collection: collection,
	}
}

func (r *PurchaseRepository) Create(ctx context.Context, purchase *models.Purchase) error {
	_, err := r.collection.InsertOne(ctx, purchase)
	return err
}

func (r *PurchaseRepository) GetByID(ctx context.Context, id string) (*models.Purchase, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var purchase models.Purchase
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&purchase)
	if err != nil {
		return nil, err
	}

	return &purchase, nil
}

func (r *PurchaseRepository) GetAll(ctx context.Context) ([]*models.Purchase, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var purchases []*models.Purchase
	for cursor.Next(ctx) {
		var purchase models.Purchase
		if err := cursor.Decode(&purchase); err != nil {
			return nil, err
		}
		purchases = append(purchases, &purchase)
	}

	return purchases, nil
}

func (r *PurchaseRepository) Update(ctx context.Context, id string, purchase *models.Purchase) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, purchase)
	return err
}

func (r *PurchaseRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// GetNextSequenceNumber gets the next sequence number for a given prefix
func (r *PurchaseRepository) GetNextSequenceNumber(ctx context.Context, prefix string) (int, error) {
	// Find the highest sequence number for this prefix
	filter := bson.M{
		"purchaseCode": bson.M{
			"$regex":   "^" + prefix,
			"$options": "i",
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "purchaseCode", Value: -1}}).SetLimit(1)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return 1, err // Start from 1 if error
	}
	defer cursor.Close(ctx)

	var lastPurchase models.Purchase
	if cursor.Next(ctx) {
		if err := cursor.Decode(&lastPurchase); err != nil {
			return 1, err
		}

		// Extract sequence number from the last purchase code
		if lastPurchase.PurchaseCode != "" {
			// Get the last 4 characters (sequence number)
			if len(lastPurchase.PurchaseCode) >= 4 {
				seqStr := lastPurchase.PurchaseCode[len(lastPurchase.PurchaseCode)-4:]
				// Try to parse as integer
				if seq, err := strconv.Atoi(seqStr); err == nil {
					return seq + 1, nil
				}
			}
		}
	}

	// If no previous purchase found or parsing failed, start from 1
	return 1, nil
}
