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

type FCLShipmentRepository struct {
	collection *mongo.Collection
}

func NewFCLShipmentRepository(collection *mongo.Collection) *FCLShipmentRepository {
	return &FCLShipmentRepository{collection: collection}
}

func (r *FCLShipmentRepository) Create(ctx context.Context, s *models.FCLShipment) error {
	result, err := r.collection.InsertOne(ctx, s)
	if err != nil {
		return err
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		s.ID = oid
	}
	return nil
}

func (r *FCLShipmentRepository) GetByID(ctx context.Context, id string) (*models.FCLShipment, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var s models.FCLShipment
	if err := r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *FCLShipmentRepository) GetAll(ctx context.Context) ([]*models.FCLShipment, error) {
	opts := options.Find().SetSort(bson.D{{Key: "shipmentDate", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var shipments []*models.FCLShipment
	for cursor.Next(ctx) {
		var s models.FCLShipment
		if err := cursor.Decode(&s); err != nil {
			return nil, err
		}
		shipments = append(shipments, &s)
	}
	return shipments, nil
}

func (r *FCLShipmentRepository) Update(ctx context.Context, id string, s *models.FCLShipment) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, s)
	return err
}

func (r *FCLShipmentRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// GetNextSequenceNumber returns the next sequence number for FCL codes with the given prefix.
func (r *FCLShipmentRepository) GetNextSequenceNumber(ctx context.Context, prefix string) (int, error) {
	filter := bson.M{
		"fclCode": bson.M{
			"$regex":   "^" + prefix,
			"$options": "i",
		},
	}
	opts := options.Find().SetSort(bson.D{{Key: "fclCode", Value: -1}}).SetLimit(1)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return 1, err
	}
	defer cursor.Close(ctx)

	var last models.FCLShipment
	if cursor.Next(ctx) {
		if err := cursor.Decode(&last); err != nil {
			return 1, err
		}
		if last.FCLCode != "" && len(last.FCLCode) >= 4 {
			seqStr := last.FCLCode[len(last.FCLCode)-4:]
			if seq, err := strconv.Atoi(seqStr); err == nil {
				return seq + 1, nil
			}
		}
	}
	return 1, nil
}
