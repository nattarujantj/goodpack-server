package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goodpack-server/models"
)

type SupplierRepository struct {
	collection *mongo.Collection
}

func NewSupplierRepository(collection *mongo.Collection) *SupplierRepository {
	return &SupplierRepository{
		collection: collection,
	}
}

func (r *SupplierRepository) Create(supplier *models.Supplier) error {
	ctx := context.Background()

	// Generate supplier code
	supplierCode, err := r.generateSupplierCode()
	if err != nil {
		return err
	}
	supplier.SupplierCode = supplierCode

	_, err = r.collection.InsertOne(ctx, supplier)
	return err
}

func (r *SupplierRepository) GetByID(id string) (*models.Supplier, error) {
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var supplier models.Supplier
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&supplier)
	if err != nil {
		return nil, err
	}

	return &supplier, nil
}

func (r *SupplierRepository) GetAll() ([]*models.Supplier, error) {
	ctx := context.Background()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var suppliers []*models.Supplier
	for cursor.Next(ctx) {
		var supplier models.Supplier
		if err := cursor.Decode(&supplier); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, &supplier)
	}

	return suppliers, nil
}

func (r *SupplierRepository) Update(id string, supplier *models.Supplier) error {
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, supplier)
	return err
}

func (r *SupplierRepository) Delete(id string) error {
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *SupplierRepository) GetBySupplierCode(supplierCode string) (*models.Supplier, error) {
	ctx := context.Background()

	var supplier models.Supplier
	err := r.collection.FindOne(ctx, bson.M{"supplierCode": supplierCode}).Decode(&supplier)
	if err != nil {
		return nil, err
	}

	return &supplier, nil
}

func (r *SupplierRepository) generateSupplierCode() (string, error) {
	ctx := context.Background()

	// Get the highest supplier code
	opts := options.Find().SetSort(bson.D{{"supplierCode", -1}}).SetLimit(1)
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	var lastSupplier models.Supplier
	if cursor.Next(ctx) {
		if err := cursor.Decode(&lastSupplier); err != nil {
			return "", err
		}
	}

	// Extract number from last supplier code
	var nextNumber int = 1
	if lastSupplier.SupplierCode != "" {
		parts := strings.Split(lastSupplier.SupplierCode, "-")
		if len(parts) == 2 {
			if num, err := strconv.Atoi(parts[1]); err == nil {
				nextNumber = num + 1
			}
		}
	}

	// Format as S-0001, S-0002, etc.
	return fmt.Sprintf("S-%04d", nextNumber), nil
}

// GenerateSupplierCode is a public method to generate supplier code
func (r *SupplierRepository) GenerateSupplierCode() (string, error) {
	return r.generateSupplierCode()
}

