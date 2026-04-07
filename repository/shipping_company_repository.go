package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"goodpack-server/models"
)

type ShippingCompanyRepository struct {
	collection *mongo.Collection
}

func NewShippingCompanyRepository(collection *mongo.Collection) *ShippingCompanyRepository {
	return &ShippingCompanyRepository{
		collection: collection,
	}
}

func (r *ShippingCompanyRepository) Create(company *models.ShippingCompany) error {
	ctx := context.Background()
	result, err := r.collection.InsertOne(ctx, company)
	if err != nil {
		return err
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		company.ID = oid
	}
	return nil
}

func (r *ShippingCompanyRepository) GetByID(id string) (*models.ShippingCompany, error) {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var company models.ShippingCompany
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&company)
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *ShippingCompanyRepository) GetAll() ([]*models.ShippingCompany, error) {
	ctx := context.Background()
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var companies []*models.ShippingCompany
	for cursor.Next(ctx) {
		var company models.ShippingCompany
		if err := cursor.Decode(&company); err != nil {
			return nil, err
		}
		companies = append(companies, &company)
	}
	return companies, nil
}

func (r *ShippingCompanyRepository) Update(id string, company *models.ShippingCompany) error {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, company)
	return err
}

func (r *ShippingCompanyRepository) Delete(id string) error {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}
