package repository

import (
	"context"
	"time"

	"goodpack-server/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type QuotationRepository struct {
	collection *mongo.Collection
}

func NewQuotationRepository(collection *mongo.Collection) *QuotationRepository {
	return &QuotationRepository{
		collection: collection,
	}
}

func (r *QuotationRepository) Create(quotation *models.Quotation) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, quotation)
	return err
}

func (r *QuotationRepository) GetByID(id string) (*models.Quotation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var quotation models.Quotation
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&quotation)
	if err != nil {
		return nil, err
	}

	return &quotation, nil
}

func (r *QuotationRepository) GetAll() ([]*models.Quotation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var quotations []*models.Quotation
	for cursor.Next(ctx) {
		var quotation models.Quotation
		if err := cursor.Decode(&quotation); err != nil {
			continue
		}
		quotations = append(quotations, &quotation)
	}

	return quotations, cursor.Err()
}

func (r *QuotationRepository) Update(id string, quotation *models.Quotation) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, quotation)
	return err
}

func (r *QuotationRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *QuotationRepository) GetByCode(code string) (*models.Quotation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var quotation models.Quotation
	err := r.collection.FindOne(ctx, bson.M{"quotationCode": code}).Decode(&quotation)
	if err != nil {
		return nil, err
	}

	return &quotation, nil
}

func (r *QuotationRepository) GetLastQuotationCode(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var quotation models.Quotation
	opts := options.FindOne().SetSort(bson.D{primitive.E{Key: "quotationCode", Value: -1}})
	err := r.collection.FindOne(ctx, bson.M{}, opts).Decode(&quotation)
	if err == mongo.ErrNoDocuments {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return quotation.QuotationCode, nil
}

func (r *QuotationRepository) GetByCustomer(customerID string) ([]*models.Quotation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"customerId": customerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var quotations []*models.Quotation
	for cursor.Next(ctx) {
		var quotation models.Quotation
		if err := cursor.Decode(&quotation); err != nil {
			continue
		}
		quotations = append(quotations, &quotation)
	}

	return quotations, cursor.Err()
}

func (r *QuotationRepository) GetByStatus(status string) ([]*models.Quotation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var quotations []*models.Quotation
	for cursor.Next(ctx) {
		var quotation models.Quotation
		if err := cursor.Decode(&quotation); err != nil {
			continue
		}
		quotations = append(quotations, &quotation)
	}

	return quotations, cursor.Err()
}
