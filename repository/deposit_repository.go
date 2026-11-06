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

type DepositRepository struct {
	collection *mongo.Collection
}

func NewDepositRepository(collection *mongo.Collection) *DepositRepository {
	return &DepositRepository{collection: collection}
}

func (r *DepositRepository) Create(ctx context.Context, deposit *models.Deposit) error {
	result, err := r.collection.InsertOne(ctx, deposit)
	if err != nil {
		return err
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		deposit.ID = oid
	}
	return nil
}

func (r *DepositRepository) GetByID(ctx context.Context, id string) (*models.Deposit, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var deposit models.Deposit
	if err := r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&deposit); err != nil {
		return nil, err
	}
	return &deposit, nil
}

func (r *DepositRepository) GetByCode(ctx context.Context, code string) (*models.Deposit, error) {
	var deposit models.Deposit
	if err := r.collection.FindOne(ctx, bson.M{"depositCode": code}).Decode(&deposit); err != nil {
		return nil, err
	}
	return &deposit, nil
}

// GetAll คืนรายการมัดจำ กรองตามประเภท ("customer"/"supplier") และ partyId ได้ (ว่าง = ไม่กรอง)
// เรียงตามวันที่วางมัดจำล่าสุดก่อน
func (r *DepositRepository) GetAll(ctx context.Context, depositType, partyID string) ([]*models.Deposit, error) {
	filter := bson.M{}
	if depositType != "" {
		filter["type"] = depositType
	}
	if partyID != "" {
		filter["partyId"] = partyID
	}
	opts := options.Find().SetSort(bson.D{{Key: "depositDate", Value: -1}, {Key: "createdAt", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	deposits := []*models.Deposit{}
	for cursor.Next(ctx) {
		var d models.Deposit
		if err := cursor.Decode(&d); err != nil {
			return nil, err
		}
		deposits = append(deposits, &d)
	}
	return deposits, cursor.Err()
}

func (r *DepositRepository) Update(ctx context.Context, id string, deposit *models.Deposit) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, deposit)
	return err
}

func (r *DepositRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// GetNextSequenceNumber หาเลขลำดับถัดไปของรหัสมัดจำที่ขึ้นต้นด้วย prefix (เช่น "DR-")
// เลขลำดับต่อเนื่องข้ามเดือน จึงหาค่าสูงสุดจากทุกรหัสที่มี prefix นี้
func (r *DepositRepository) GetNextSequenceNumber(ctx context.Context, prefix string) (int, error) {
	filter := bson.M{"depositCode": bson.M{"$regex": "^" + prefix}}
	opts := options.Find().SetProjection(bson.M{"depositCode": 1})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	maxSeq := 0
	for cursor.Next(ctx) {
		var d struct {
			DepositCode string `bson:"depositCode"`
		}
		if err := cursor.Decode(&d); err != nil {
			continue
		}
		parts := strings.Split(d.DepositCode, "-")
		if seq, err := strconv.Atoi(parts[len(parts)-1]); err == nil && seq > maxSeq {
			maxSeq = seq
		}
	}
	return maxSeq + 1, nil
}
