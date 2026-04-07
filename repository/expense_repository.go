package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goodpack-server/models"
)

type ExpenseRepository struct {
	collection *mongo.Collection
}

func NewExpenseRepository(collection *mongo.Collection) *ExpenseRepository {
	return &ExpenseRepository{
		collection: collection,
	}
}

func (r *ExpenseRepository) Create(ctx context.Context, expense *models.Expense) error {
	result, err := r.collection.InsertOne(ctx, expense)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		expense.ID = oid
	}

	return nil
}

func (r *ExpenseRepository) GetByID(ctx context.Context, id string) (*models.Expense, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var expense models.Expense
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&expense)
	if err != nil {
		return nil, err
	}

	return &expense, nil
}

func (r *ExpenseRepository) GetAll(ctx context.Context) ([]*models.Expense, error) {
	opts := options.Find().SetSort(bson.D{{Key: "expenseDate", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var expenses []*models.Expense
	for cursor.Next(ctx) {
		var expense models.Expense
		if err := cursor.Decode(&expense); err != nil {
			return nil, err
		}
		expenses = append(expenses, &expense)
	}

	return expenses, nil
}

func (r *ExpenseRepository) Update(ctx context.Context, id string, expense *models.Expense) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, expense)
	return err
}

func (r *ExpenseRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}
