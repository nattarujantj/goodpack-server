package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProductSnapshotItem captures stock state of a single product at snapshot time
type ProductSnapshotItem struct {
	ProductID       string `bson:"productId" json:"productId"`
	SKUID           string `bson:"skuId" json:"skuId"`
	Code            string `bson:"code" json:"code"`
	Name            string `bson:"name" json:"name"`
	Category        string `bson:"category" json:"category"`
	Color           string `bson:"color" json:"color"`
	Size            string `bson:"size" json:"size"`
	VATRemaining    int    `bson:"vatRemaining" json:"vatRemaining"`
	NonVATRemaining int    `bson:"nonVATRemaining" json:"nonVATRemaining"`
	ActualStock     int    `bson:"actualStock" json:"actualStock"`
}

// InventorySnapshot stores the monthly inventory state for all products
type InventorySnapshot struct {
	ID               primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	Month            int                   `bson:"month" json:"month"`
	Year             int                   `bson:"year" json:"year"`
	SnapshotDate     time.Time             `bson:"snapshotDate" json:"snapshotDate"`
	CreatedBy        string                `bson:"createdBy" json:"createdBy"` // "system" or user ID
	IsManual         bool                  `bson:"isManual" json:"isManual"`
	Products         []ProductSnapshotItem `bson:"products" json:"products"`
	TotalProducts    int                   `bson:"totalProducts" json:"totalProducts"`
	TotalVATStock    int                   `bson:"totalVATStock" json:"totalVATStock"`
	TotalNonVATStock int                   `bson:"totalNonVATStock" json:"totalNonVATStock"`
	TotalActualStock int                   `bson:"totalActualStock" json:"totalActualStock"`
}

// SnapshotUpdateRequest is the body for PUT /inventory-snapshots/{year}/{month}
type SnapshotUpdateRequest struct {
	Products []ProductSnapshotItem `json:"products"`
}
