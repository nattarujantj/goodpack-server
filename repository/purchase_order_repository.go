package repository

import (
	"context"
	"fmt"
	"time"

	"goodpack-server/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PurchaseOrderRepository struct {
	collection *mongo.Collection
}

func NewPurchaseOrderRepository(collection *mongo.Collection) *PurchaseOrderRepository {
	return &PurchaseOrderRepository{
		collection: collection,
	}
}

func (r *PurchaseOrderRepository) Create(po *models.PurchaseOrder) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.collection.InsertOne(ctx, po)
	if err != nil {
		return err
	}

	// Update purchase order ID with the inserted ID
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		po.ID = oid
	}

	return nil
}

func (r *PurchaseOrderRepository) GetByID(id string) (*models.PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var po models.PurchaseOrder
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&po)
	if err != nil {
		return nil, err
	}

	return &po, nil
}

func (r *PurchaseOrderRepository) GetAll() ([]*models.PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var purchaseOrders []*models.PurchaseOrder
	for cursor.Next(ctx) {
		var po models.PurchaseOrder
		if err := cursor.Decode(&po); err != nil {
			continue
		}
		purchaseOrders = append(purchaseOrders, &po)
	}

	return purchaseOrders, cursor.Err()
}

func (r *PurchaseOrderRepository) Update(id string, po *models.PurchaseOrder) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, po)
	return err
}

func (r *PurchaseOrderRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *PurchaseOrderRepository) GetByCode(code string) (*models.PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var po models.PurchaseOrder
	err := r.collection.FindOne(ctx, bson.M{"purchaseOrderCode": code}).Decode(&po)
	if err != nil {
		return nil, err
	}

	return &po, nil
}

func (r *PurchaseOrderRepository) GetLastPurchaseOrderCode(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var po models.PurchaseOrder
	opts := options.FindOne().SetSort(bson.D{primitive.E{Key: "purchaseOrderCode", Value: -1}})
	err := r.collection.FindOne(ctx, bson.M{}, opts).Decode(&po)
	if err == mongo.ErrNoDocuments {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return po.PurchaseOrderCode, nil
}

func (r *PurchaseOrderRepository) GetBySupplier(supplierID string) ([]*models.PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"supplierId": supplierID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var purchaseOrders []*models.PurchaseOrder
	for cursor.Next(ctx) {
		var po models.PurchaseOrder
		if err := cursor.Decode(&po); err != nil {
			continue
		}
		purchaseOrders = append(purchaseOrders, &po)
	}

	return purchaseOrders, cursor.Err()
}

func (r *PurchaseOrderRepository) GetByStatus(status string) ([]*models.PurchaseOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var purchaseOrders []*models.PurchaseOrder
	for cursor.Next(ctx) {
		var po models.PurchaseOrder
		if err := cursor.Decode(&po); err != nil {
			continue
		}
		purchaseOrders = append(purchaseOrders, &po)
	}

	return purchaseOrders, cursor.Err()
}

func (r *PurchaseOrderRepository) GetNextSequenceNumber(ctx context.Context, prefix string) (int, error) {
	lastCode, err := r.GetLastPurchaseOrderCode(ctx)
	if err != nil {
		return 0, err
	}

	if lastCode == "" {
		return 1, nil
	}

	// Extract sequence number from code (e.g., PO-2412-0001 -> 1)
	var lastYear, lastMonth, lastSeq int
	_, err = fmt.Sscanf(lastCode, "PO-%02d%02d-%04d", &lastYear, &lastMonth, &lastSeq)
	if err != nil {
		return 0, err
	}

	return lastSeq + 1, nil
}

