package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionName = "products"
)

// productRepository is the MongoDB adapter implementing ports.ProductRepository
type productRepository struct {
	collection *mongo.Collection
}

// NewProductRepository creates a new MongoDB product repository
func NewProductRepository(db *mongo.Database) ports.ProductRepository {
	return &productRepository{
		collection: db.Collection(collectionName),
	}
}

// Create inserts a new product into MongoDB
func (r *productRepository) Create(ctx context.Context, product *model.Product) (*model.Product, error) {
	// Set timestamps
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now
	product.IsActive = true

	result, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return nil, fmt.Errorf("error creating product: %w", err)
	}

	// Set the generated ID back on the product
	product.ID = result.InsertedID.(primitive.ObjectID)
	return product, nil
}

// GetByID fetches a product by its ObjectID
func (r *productRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	// Convert string ID to MongoDB ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid product id: %s", id)
	}

	filter := bson.M{
		"_id":       objectID,
		"is_active": true,
	}

	var product model.Product
	err = r.collection.FindOne(ctx, filter).Decode(&product)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching product: %w", err)
	}

	return &product, nil
}

// GetBySlug fetches a product by its URL slug
func (r *productRepository) GetBySlug(ctx context.Context, slug string) (*model.Product, error) {
	filter := bson.M{
		"slug":      slug,
		"is_active": true,
	}

	var product model.Product
	err := r.collection.FindOne(ctx, filter).Decode(&product)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching product: %w", err)
	}

	return &product, nil
}

// List fetches products with filters and cursor-based pagination
func (r *productRepository) List(ctx context.Context, filter model.ProductFilter) (*model.ProductListResponse, error) {
	// Build MongoDB filter
	mongoFilter := bson.M{"is_active": true}

	if filter.Category != "" {
		mongoFilter["category"] = filter.Category
	}
	if filter.Brand != "" {
		mongoFilter["brand"] = filter.Brand
	}
	if filter.MinPrice > 0 || filter.MaxPrice > 0 {
		priceFilter := bson.M{}
		if filter.MinPrice > 0 {
			priceFilter["$gte"] = filter.MinPrice
		}
		if filter.MaxPrice > 0 {
			priceFilter["$lte"] = filter.MaxPrice
		}
		mongoFilter["base_price"] = priceFilter
	}

	// Cursor-based pagination
	// If cursor provided, only fetch products AFTER that cursor
	if filter.Cursor != "" {
		cursorID, err := primitive.ObjectIDFromHex(filter.Cursor)
		if err == nil {
			mongoFilter["_id"] = bson.M{"$gt": cursorID}
		}
	}

	// Set limit (default 20, max 100)
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Count total matching documents
	total, err := r.collection.CountDocuments(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, fmt.Errorf("error counting products: %w", err)
	}

	// Fetch products
	findOptions := options.Find().
		SetSort(bson.M{"_id": 1}).
		SetLimit(limit + 1) // fetch one extra to know if there's a next page

	cursor, err := r.collection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error fetching products: %w", err)
	}
	defer cursor.Close(ctx)

	var products []model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, fmt.Errorf("error decoding products: %w", err)
	}

	// Determine next cursor
	var nextCursor string
	if int64(len(products)) > limit {
		// There are more products
		products = products[:limit]                     // remove the extra one
		nextCursor = products[len(products)-1].ID.Hex() // cursor = last item's ID
	}

	return &model.ProductListResponse{
		Products:   products,
		NextCursor: nextCursor,
		Total:      total,
	}, nil
}

// Search performs full-text search on products
func (r *productRepository) Search(ctx context.Context, query string, limit int64) ([]model.Product, error) {
	if limit <= 0 {
		limit = 20
	}

	filter := bson.M{
		"$text":     bson.M{"$search": query},
		"is_active": true,
	}

	// Sort by text search score (most relevant first)
	findOptions := options.Find().
		SetSort(bson.M{"score": bson.M{"$meta": "textScore"}}).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error searching products: %w", err)
	}
	defer cursor.Close(ctx)

	var products []model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, fmt.Errorf("error decoding search results: %w", err)
	}

	return products, nil
}

// Update updates an existing product
func (r *productRepository) Update(ctx context.Context, id string, req *model.UpdateProductRequest) (*model.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid product id: %s", id)
	}

	// Build update document
	updateFields := bson.M{
		"updated_at": time.Now(),
	}

	if req.Name != "" {
		updateFields["name"] = req.Name
	}
	if req.Description != "" {
		updateFields["description"] = req.Description
	}
	if req.Brand != "" {
		updateFields["brand"] = req.Brand
	}
	if req.BasePrice > 0 {
		updateFields["base_price"] = req.BasePrice
	}
	if req.Variants != nil {
		updateFields["variants"] = req.Variants
	}
	if req.Attributes != nil {
		updateFields["attributes"] = req.Attributes
	}
	if req.IsActive != nil {
		updateFields["is_active"] = *req.IsActive
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": updateFields}

	// FindOneAndUpdate returns the updated document
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedProduct model.Product
	err = r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedProduct)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error updating product: %w", err)
	}

	return &updatedProduct, nil
}

// Delete soft-deletes a product (sets is_active = false)
func (r *productRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid product id: %s", id)
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{
		"is_active":  false,
		"updated_at": time.Now(),
	}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}
