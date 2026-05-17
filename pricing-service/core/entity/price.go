package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Price represents the pricing information.
type Price struct {
	ID          string               `json:"id" bson:"_id,omitempty"`
	PriceType   PriceType            `json:"priceType" bson:"priceType"`
	Value       primitive.Decimal128 `json:"value" bson:"value"`
	Description string               `json:"description" bson:"description"`
	CreatedAt   time.Time            `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt" bson:"updatedAt"`
}
