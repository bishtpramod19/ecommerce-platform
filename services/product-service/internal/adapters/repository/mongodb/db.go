package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoDBClient creates a new MongoDB client and returns the database.
func NewMongoDBClient(cfg *config.Config) (*mongo.Database, error) {
	// Set connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure client options
	clientOptions := options.Client().
		ApplyURI(cfg.MongoURI).
		SetMaxPoolSize(50).
		SetMinPoolSize(5).
		SetMaxConnIdleTime(30 * time.Minute)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("error connecting to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("error pinging MongoDB: %w", err)
	}

	log.Println("Successfully connected to MongoDB")

	// Get database
	db := client.Database(cfg.MongoDB)

	// Create indexes
	if err := createIndexes(ctx, db); err != nil {
		return nil, fmt.Errorf("error creating indexes: %w", err)
	}

	return db, nil
}

// createIndexes creates all required indexes for the products collection.
// Indexes are created on startup — idempotent (safe to run multiple times).
func createIndexes(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection(collectionName)

	indexes := []mongo.IndexModel{
		// Text search index (name, description, brand)
		// Weights determine relevance: name match > brand > description
		{
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "description", Value: "text"},
				{Key: "brand", Value: "text"},
			},
			Options: options.Index().SetWeights(bson.M{
				"name":        10,
				"brand":       5,
				"description": 1,
			}).SetName("product_text_search"),
		},

		// Category index (most common filter)
		{
			Keys:    bson.M{"category": 1},
			Options: options.Index().SetName("category_idx"),
		},

		// Category + price compound index (filter + sort)
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "base_price", Value: 1},
			},
			Options: options.Index().SetName("category_price_idx"),
		},

		// is_active index (always filter by this)
		{
			Keys:    bson.M{"is_active": 1},
			Options: options.Index().SetName("is_active_idx"),
		},

		// Slug unique index (for SEO URLs)
		{
			Keys:    bson.M{"slug": 1},
			Options: options.Index().SetUnique(true).SetName("slug_unique_idx"),
		},

		// Trending score index (for trending products)
		{
			Keys:    bson.M{"metadata.trending_score": -1},
			Options: options.Index().SetName("trending_score_idx"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("error creating indexes: %w", err)
	}

	log.Println("MongoDB indexes created successfully")
	return nil
}
