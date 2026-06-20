package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"service-app-go/pricing-service/core/entity"
)

const (
	collectionName = "prices"
)

// MongoPriceRepository implements PriceRepository for MongoDB.
type MongoPriceRepository struct {
	collection *mongo.Collection
}

// NewMongoPriceRepository creates a new MongoPriceRepository.
func NewMongoPriceRepository(client *mongo.Client, databaseName string) *MongoPriceRepository {
	collection := client.Database(databaseName).Collection(collectionName)
	return &MongoPriceRepository{
		collection: collection,
	}
}

// Save inserts a new price or updates an existing one (upsert by _id).
func (r *MongoPriceRepository) Save(ctx context.Context, price entity.Price) (*entity.Price, error) {
	now := time.Now()
	if price.CreatedAt.IsZero() {
		price.CreatedAt = now
	}
	price.UpdatedAt = now

	filter := bson.M{"_id": price.ID}
	update := bson.M{"$set": price}
	opts := options.Update().SetUpsert(true)

	if _, err := r.collection.UpdateOne(ctx, filter, update, opts); err != nil {
		return nil, fmt.Errorf("failed to save price: %w", err)
	}
	return &price, nil
}

// FindByPriceType retrieves a price by its PriceType.
func (r *MongoPriceRepository) FindByPriceType(ctx context.Context, priceType entity.PriceType) (*entity.Price, error) {
	var price entity.Price
	filter := bson.M{"priceType": priceType}
	err := r.collection.FindOne(ctx, filter).Decode(&price)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find price by PriceType %s: %w", priceType, err)
	}
	return &price, nil
}

// FindAll retrieves all prices.
func (r *MongoPriceRepository) FindAll(ctx context.Context) ([]entity.Price, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find all prices: %w", err)
	}
	defer cursor.Close(ctx)

	var prices []entity.Price
	if err := cursor.All(ctx, &prices); err != nil {
		return nil, fmt.Errorf("failed to decode prices: %w", err)
	}
	return prices, nil
}
