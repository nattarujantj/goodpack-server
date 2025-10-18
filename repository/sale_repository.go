package repository

import (
	"context"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goodpack-server/models"
)

type SaleRepository struct {
	collection *mongo.Collection
}

func NewSaleRepository(collection *mongo.Collection) *SaleRepository {
	return &SaleRepository{
		collection: collection,
	}
}

func (r *SaleRepository) Create(sale *models.Sale) error {
	ctx := context.Background()
	_, err := r.collection.InsertOne(ctx, sale)
	return err
}

func (r *SaleRepository) GetByID(id string) (*models.Sale, error) {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var sale models.Sale
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&sale)
	if err != nil {
		return nil, err
	}

	return &sale, nil
}

func (r *SaleRepository) GetAll(ctx context.Context) ([]models.Sale, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sales []models.Sale
	if err = cursor.All(ctx, &sales); err != nil {
		return nil, err
	}

	return sales, nil
}

func (r *SaleRepository) Update(id string, sale *models.Sale) error {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, sale)
	return err
}

func (r *SaleRepository) Delete(id string) error {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *SaleRepository) GetNextSequenceNumber(ctx context.Context, prefix string) (int, error) {
	// Find the highest sequence number for the given prefix
	filter := bson.M{
		"saleCode": bson.M{
			"$regex":   "^" + prefix,
			"$options": "i",
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "saleCode", Value: -1}}).SetLimit(1)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var lastSale models.Sale
	if cursor.Next(ctx) {
		if err := cursor.Decode(&lastSale); err != nil {
			return 0, err
		}
	}

	// Extract sequence number from saleCode
	if lastSale.SaleCode == "" {
		return 1, nil
	}

	// Extract the last 4 digits from saleCode
	parts := strings.Split(lastSale.SaleCode, "-")
	if len(parts) < 3 {
		return 1, nil
	}

	seqStr := parts[len(parts)-1]
	seq, err := strconv.Atoi(seqStr)
	if err != nil {
		return 1, nil
	}

	return seq + 1, nil
}
