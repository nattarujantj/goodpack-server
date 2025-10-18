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

type CustomerRepository struct {
	collection *mongo.Collection
}

func NewCustomerRepository(collection *mongo.Collection) *CustomerRepository {
	return &CustomerRepository{
		collection: collection,
	}
}

func (r *CustomerRepository) Create(customer *models.Customer) error {
	ctx := context.Background()

	// Generate customer code
	customerCode, err := r.generateCustomerCode()
	if err != nil {
		return err
	}
	customer.CustomerCode = customerCode

	_, err = r.collection.InsertOne(ctx, customer)
	return err
}

func (r *CustomerRepository) GetByID(id string) (*models.Customer, error) {
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var customer models.Customer
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&customer)
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (r *CustomerRepository) GetAll() ([]*models.Customer, error) {
	ctx := context.Background()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var customers []*models.Customer
	for cursor.Next(ctx) {
		var customer models.Customer
		if err := cursor.Decode(&customer); err != nil {
			return nil, err
		}
		customers = append(customers, &customer)
	}

	return customers, nil
}

func (r *CustomerRepository) Update(id string, customer *models.Customer) error {
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, customer)
	return err
}

func (r *CustomerRepository) Delete(id string) error {
	ctx := context.Background()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *CustomerRepository) GetByCustomerCode(customerCode string) (*models.Customer, error) {
	ctx := context.Background()

	var customer models.Customer
	err := r.collection.FindOne(ctx, bson.M{"customerCode": customerCode}).Decode(&customer)
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (r *CustomerRepository) generateCustomerCode() (string, error) {
	ctx := context.Background()

	// Get the highest customer code
	opts := options.Find().SetSort(bson.D{{"customerCode", -1}}).SetLimit(1)
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	var lastCustomer models.Customer
	if cursor.Next(ctx) {
		if err := cursor.Decode(&lastCustomer); err != nil {
			return "", err
		}
	}

	// Extract number from last customer code
	var nextNumber int = 1
	if lastCustomer.CustomerCode != "" {
		parts := strings.Split(lastCustomer.CustomerCode, "-")
		if len(parts) == 2 {
			if num, err := strconv.Atoi(parts[1]); err == nil {
				nextNumber = num + 1
			}
		}
	}

	// Format as C-0001, C-0002, etc.
	return fmt.Sprintf("C-%04d", nextNumber), nil
}
