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

type InternationalImportRepository struct {
	collection *mongo.Collection
}

func NewInternationalImportRepository(collection *mongo.Collection) *InternationalImportRepository {
	return &InternationalImportRepository{
		collection: collection,
	}
}

func (r *InternationalImportRepository) Create(ctx context.Context, imp *models.InternationalImport) error {
	result, err := r.collection.InsertOne(ctx, imp)
	if err != nil {
		return err
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		imp.ID = oid
	}
	return nil
}

func (r *InternationalImportRepository) GetByID(ctx context.Context, id string) (*models.InternationalImport, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var imp models.InternationalImport
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&imp)
	if err != nil {
		return nil, err
	}
	return &imp, nil
}

func (r *InternationalImportRepository) GetAll(ctx context.Context) ([]*models.InternationalImport, error) {
	opts := options.Find().SetSort(bson.D{{Key: "importDate", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var imports []*models.InternationalImport
	for cursor.Next(ctx) {
		var imp models.InternationalImport
		if err := cursor.Decode(&imp); err != nil {
			return nil, err
		}
		imports = append(imports, &imp)
	}
	return imports, nil
}

func (r *InternationalImportRepository) Update(ctx context.Context, id string, imp *models.InternationalImport) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, imp)
	return err
}

func (r *InternationalImportRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// GetByFCLShipmentID returns all imports linked to the given FCL container.
func (r *InternationalImportRepository) GetByFCLShipmentID(ctx context.Context, shipmentID string) ([]*models.InternationalImport, error) {
	opts := options.Find().SetSort(bson.D{{Key: "importDate", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"fclShipmentId": shipmentID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var imports []*models.InternationalImport
	for cursor.Next(ctx) {
		var imp models.InternationalImport
		if err := cursor.Decode(&imp); err != nil {
			return nil, err
		}
		imports = append(imports, &imp)
	}
	return imports, nil
}

// GetNextSequenceNumber returns the next sequence number for import codes with the given prefix.
func (r *InternationalImportRepository) GetNextSequenceNumber(ctx context.Context, prefix string) (int, error) {
	filter := bson.M{
		"importCode": bson.M{
			"$regex":   "^" + prefix,
			"$options": "i",
		},
	}
	opts := options.Find().SetSort(bson.D{{Key: "importCode", Value: -1}}).SetLimit(1)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return 1, err
	}
	defer cursor.Close(ctx)

	var last models.InternationalImport
	if cursor.Next(ctx) {
		if err := cursor.Decode(&last); err != nil {
			return 1, err
		}
		if last.ImportCode != "" && len(last.ImportCode) >= 4 {
			seqStr := last.ImportCode[len(last.ImportCode)-4:]
			if seq, err := strconv.Atoi(seqStr); err == nil {
				return seq + 1, nil
			}
		}
	}
	return 1, nil
}
