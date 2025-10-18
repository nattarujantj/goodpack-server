package repository

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goodpack-server/models"
	"goodpack-server/utils"
)

type ProductRepository struct {
	collection   *mongo.Collection
	skuGenerator *utils.SKUGenerator
}

func NewProductRepository(collection *mongo.Collection) *ProductRepository {
	return &ProductRepository{
		collection:   collection,
		skuGenerator: utils.NewSKUGenerator(),
	}
}

func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
	// Generate SKU ID
	existingSKUs, err := r.getAllSKUIDs(ctx)
	if err != nil {
		return err
	}

	nextNumber := r.skuGenerator.GetNextSKUNumber(product.Category, existingSKUs)
	product.SKUID = r.skuGenerator.GenerateSKUID(product.Category, nextNumber)

	// Generate Product Code
	product.Code = r.skuGenerator.GenerateProductCode(product.Category, product.Size, product.Color)

	// Generate QR Data
	product.QRData = product.SKUID // Use SKU ID as QR data

	_, err = r.collection.InsertOne(ctx, product)
	return err
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var product models.Product
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *ProductRepository) GetAll(ctx context.Context) ([]*models.Product, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*models.Product
	for cursor.Next(ctx) {
		var product models.Product
		if err := cursor.Decode(&product); err != nil {
			log.Printf("Error decoding product: %v", err)
			continue
		}
		products = append(products, &product)
	}

	return products, cursor.Err()
}

func (r *ProductRepository) Update(ctx context.Context, id string, product *models.Product) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, product)
	return err
}

func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *ProductRepository) UpdateStock(ctx context.Context, id string, stock models.Stock) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"stock":     stock,
				"updatedAt": time.Now(),
			},
		},
	)
	return err
}

func (r *ProductRepository) UpdatePrice(ctx context.Context, id string, price models.Price) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$set": bson.M{
				"price":     price,
				"updatedAt": time.Now(),
			},
		},
	)
	return err
}

func (r *ProductRepository) GetBySKUID(ctx context.Context, skuID string) (*models.Product, error) {
	var product models.Product
	err := r.collection.FindOne(ctx, bson.M{"skuId": skuID}).Decode(&product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *ProductRepository) GetByCode(ctx context.Context, code string) (*models.Product, error) {
	var product models.Product
	err := r.collection.FindOne(ctx, bson.M{"code": code}).Decode(&product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *ProductRepository) GetByCategory(ctx context.Context, category string) ([]*models.Product, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"category": category})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*models.Product
	for cursor.Next(ctx) {
		var product models.Product
		if err := cursor.Decode(&product); err != nil {
			log.Printf("Error decoding product: %v", err)
			continue
		}
		products = append(products, &product)
	}

	return products, cursor.Err()
}

func (r *ProductRepository) GetLowStockProducts(ctx context.Context, threshold int) ([]*models.Product, error) {
	// This would need to be implemented with aggregation pipeline
	// For now, we'll get all products and filter in memory
	allProducts, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var lowStockProducts []*models.Product
	for _, product := range allProducts {
		if product.GetTotalStock() <= threshold {
			lowStockProducts = append(lowStockProducts, product)
		}
	}

	return lowStockProducts, nil
}

// getAllSKUIDs gets all existing SKU IDs for number generation
func (r *ProductRepository) getAllSKUIDs(ctx context.Context) ([]string, error) {
	opts := options.Find().SetProjection(bson.M{"skuId": 1})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var skuIDs []string
	for cursor.Next(ctx) {
		var result struct {
			SKUID string `bson:"skuId"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding SKU ID: %v", err)
			continue
		}
		skuIDs = append(skuIDs, result.SKUID)
	}

	return skuIDs, cursor.Err()
}

func (r *ProductRepository) GetCategories(ctx context.Context) ([]string, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"category": bson.M{"$ne": nil}}}},
		{{"$group", bson.M{"_id": "$category"}}},
		{{"$sort", bson.M{"_id": 1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var categories []string
	for cursor.Next(ctx) {
		var result struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding category: %v", err)
			continue
		}
		categories = append(categories, result.ID)
	}

	return categories, cursor.Err()
}
