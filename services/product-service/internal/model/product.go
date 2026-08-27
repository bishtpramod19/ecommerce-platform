package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Variant represents a product variant (size, color, etc.)
// Example: Red T-Shirt in Size M
type Variant struct {
	SKU      string   `json:"sku"       bson:"sku"`
	Size     string   `json:"size"      bson:"size"`
	Color    string   `json:"color"     bson:"color"`
	ColorHex string   `json:"color_hex" bson:"color_hex"`
	Price    float64  `json:"price"     bson:"price"`
	Images   []string `json:"images"    bson:"images"`
}

// Metadata holds additional product information
type Metadata struct {
	Tags           []string `json:"tags"            bson:"tags"`
	SearchKeywords []string `json:"search_keywords" bson:"search_keywords"`
	AvgRating      float64  `json:"avg_rating"      bson:"avg_rating"`
	ReviewCount    int64    `json:"review_count"    bson:"review_count"`
	SoldCount      int64    `json:"sold_count"      bson:"sold_count"`
	TrendingScore  float64  `json:"trending_score"  bson:"trending_score"`
}

// Product represents a product in our catalog.
// Stored in MongoDB as a document.
type Product struct {
	ID          primitive.ObjectID `json:"id"           bson:"_id,omitempty"`
	Name        string             `json:"name"         bson:"name"`
	Slug        string             `json:"slug"         bson:"slug"`
	Description string             `json:"description"  bson:"description"`
	Brand       string             `json:"brand"        bson:"brand"`
	Category    string             `json:"category"     bson:"category"`
	SubCategory string             `json:"sub_category" bson:"sub_category"`
	BasePrice   float64            `json:"base_price"   bson:"base_price"`
	Currency    string             `json:"currency"     bson:"currency"`
	Variants    []Variant          `json:"variants"     bson:"variants"`
	Attributes  map[string]string  `json:"attributes"   bson:"attributes"`
	Metadata    Metadata           `json:"metadata"     bson:"metadata"`
	IsActive    bool               `json:"is_active"    bson:"is_active"`
	CreatedBy   string             `json:"created_by"   bson:"created_by"`
	CreatedAt   time.Time          `json:"created_at"   bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"   bson:"updated_at"`
}

// CreateProductRequest is what admin sends to create a product
type CreateProductRequest struct {
	Name        string            `json:"name"         validate:"required"`
	Description string            `json:"description"  validate:"required"`
	Brand       string            `json:"brand"        validate:"required"`
	Category    string            `json:"category"     validate:"required"`
	SubCategory string            `json:"sub_category"`
	BasePrice   float64           `json:"base_price"   validate:"required,gt=0"`
	Currency    string            `json:"currency"     validate:"required"`
	Variants    []Variant         `json:"variants"     validate:"required,min=1"`
	Attributes  map[string]string `json:"attributes"`
	Metadata    Metadata          `json:"metadata"`
}

// UpdateProductRequest is what admin sends to update a product
type UpdateProductRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Brand       string            `json:"brand"`
	BasePrice   float64           `json:"base_price"  validate:"omitempty,gt=0"`
	Variants    []Variant         `json:"variants"`
	Attributes  map[string]string `json:"attributes"`
	Metadata    Metadata          `json:"metadata"`
	IsActive    *bool             `json:"is_active"`
}

// ProductFilter holds query parameters for listing products
type ProductFilter struct {
	Category    string  `json:"category"`
	Brand       string  `json:"brand"`
	MinPrice    float64 `json:"min_price"`
	MaxPrice    float64 `json:"max_price"`
	SearchQuery string  `json:"search"`
	Cursor      string  `json:"cursor"` // for cursor-based pagination
	Limit       int64   `json:"limit"`
}

// ProductListResponse is the paginated response for product listing
type ProductListResponse struct {
	Products   []Product `json:"products"`
	NextCursor string    `json:"next_cursor"` // empty if no more pages
	Total      int64     `json:"total"`
}
