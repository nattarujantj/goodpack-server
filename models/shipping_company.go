package models

import (
	"time"

	"goodpack-server/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ShippingCompany struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type ShippingCompanyRequest struct {
	Name string `json:"name"`
}

func (r *ShippingCompanyRequest) ToShippingCompany() *ShippingCompany {
	now := utils.NowInThailand()
	return &ShippingCompany{
		Name:      r.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *ShippingCompany) UpdateFromRequest(r *ShippingCompanyRequest) {
	s.Name = r.Name
	s.UpdatedAt = utils.NowInThailand()
}
