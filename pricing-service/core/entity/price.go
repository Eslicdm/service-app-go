package entity

import (
	"time"
)

// Price represents the pricing information stored in MongoDB (collection "prices").
// Value uses float64 for Go-idiomatic handling (the DTOs also use float64);
// Mongo stores it as a 64-bit float. This matches the Spring reference's
// BigDecimal value at the wire level (JSON number).
type Price struct {
	ID          string    `json:"id" bson:"_id,omitempty"`
	PriceType   PriceType `json:"priceType" bson:"priceType"`
	Value       float64   `json:"value" bson:"value"`
	Description string    `json:"description" bson:"description"`
	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" bson:"updatedAt"`
}
