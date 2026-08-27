package ports

import (
	"context"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/model"
)

type ProductRepository interface {
	// Create inserts a new product and returns created product with ID.
	Create(ctx context.Context, product *model.Product) (*model.Product, error)

	// GetByID fetches a product by its MongoDB ObjectID.
	GetByID(ctx context.Context, id string) (*model.Product, error)

	// GetBySlug fetches a product by its URL slug.
	GetBySlug(ctx context.Context, slug string) (*model.Product, error)

	// List fetches products with filters and cursor-based pagination.
	List(ctx context.Context, filter model.ProductFilter) (*model.ProductListResponse, error)

	// Search performs full-text search on products.
	Search(ctx context.Context, query string, limit int64) ([]model.Product, error)

	// Update updates an existing product.
	Update(ctx context.Context, id string, req *model.UpdateProductRequest) (*model.Product, error)

	// Delete soft-deletes a product (sets is_active = false).
	Delete(ctx context.Context, id string) error
}
