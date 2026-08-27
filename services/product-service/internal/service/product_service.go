package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/ports"
)

const (
	// Cache TTLs
	productCacheTTL     = 1 * time.Hour
	productListCacheTTL = 5 * time.Minute

	// Cache key prefixes
	productCachePrefix     = "product:"
	productListCachePrefix = "products:list:"
	productSearchPrefix    = "products:search:"
)

// ProductService handles all business logic for products.
// Depends only on PORTS (interfaces), never on concrete implementations.
type ProductService struct {
	productStore ports.ProductRepository
	productCache ports.ProductCache
}

// NewProductService creates a new ProductService.
func NewProductService(productStore ports.ProductRepository, productCache ports.ProductCache) *ProductService {
	return &ProductService{
		productStore: productStore,
		productCache: productCache,
	}
}

// CreateProduct creates a new product.
func (s *ProductService) CreateProduct(ctx context.Context, req *model.CreateProductRequest, createdBy string) (*model.Product, error) {
	// Generate URL-friendly slug from name
	slug := generateSlug(req.Name)

	product := &model.Product{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Brand:       req.Brand,
		Category:    req.Category,
		SubCategory: req.SubCategory,
		BasePrice:   req.BasePrice,
		Currency:    req.Currency,
		Variants:    req.Variants,
		Attributes:  req.Attributes,
		Metadata:    req.Metadata,
		IsActive:    true,
		CreatedBy:   createdBy,
	}

	createdProduct, err := s.productStore.Create(ctx, product)
	if err != nil {
		return nil, fmt.Errorf("error creating product: %w", err)
	}

	// Invalidate list caches since new product added
	// We don't know which list caches are affected → invalidate all
	_ = s.productCache.DeleteByPattern(ctx, productListCachePrefix+"*")

	return createdProduct, nil
}

// GetProduct fetches a single product by ID.
// Implements cache-aside pattern:
//  1. Check cache
//  2. If miss → fetch from DB
//  3. Store in cache
//  4. Return
func (s *ProductService) GetProduct(ctx context.Context, id string) (*model.Product, error) {
	cacheKey := productCachePrefix + id

	// Step 1: Check cache
	cached, err := s.productCache.Get(ctx, cacheKey)
	if err != nil {
		// Cache error → log and continue (don't fail request because of cache!)
		fmt.Printf("cache error for key %s: %v\n", cacheKey, err)
	}
	if cached != nil {
		// Cache hit → return immediately
		return cached, nil
	}

	// Step 2: Cache miss → fetch from MongoDB
	product, err := s.productStore.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error fetching product: %w", err)
	}

	// Step 3: Store in cache for next request
	if err := s.productCache.Set(ctx, cacheKey, product, productCacheTTL); err != nil {
		// Cache error → log but don't fail (cache is optional, not critical)
		fmt.Printf("error caching product %s: %v\n", id, err)
	}

	return product, nil
}

// ListProducts fetches products with filters and cursor-based pagination.
func (s *ProductService) ListProducts(ctx context.Context, filter model.ProductFilter) (*model.ProductListResponse, error) {
	return s.productStore.List(ctx, filter)
}

// SearchProducts performs full-text search.
func (s *ProductService) SearchProducts(ctx context.Context, query string, limit int64) ([]model.Product, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	return s.productStore.Search(ctx, query, limit)
}

// UpdateProduct updates an existing product.
func (s *ProductService) UpdateProduct(ctx context.Context, id string, req *model.UpdateProductRequest) (*model.Product, error) {
	updatedProduct, err := s.productStore.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("error updating product: %w", err)
	}

	// Invalidate cache for this specific product
	cacheKey := productCachePrefix + id
	_ = s.productCache.Delete(ctx, cacheKey)

	// Invalidate list caches (product data changed)
	_ = s.productCache.DeleteByPattern(ctx, productListCachePrefix+"*")
	_ = s.productCache.DeleteByPattern(ctx, productSearchPrefix+"*")

	return updatedProduct, nil
}

// DeleteProduct soft-deletes a product.
func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	if err := s.productStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}

	// Invalidate all related caches
	_ = s.productCache.Delete(ctx, productCachePrefix+id)
	_ = s.productCache.DeleteByPattern(ctx, productListCachePrefix+"*")
	_ = s.productCache.DeleteByPattern(ctx, productSearchPrefix+"*")

	return nil
}

// generateSlug converts a product name to a URL-friendly slug.
// Example: "Cotton Round Neck T-Shirt" → "cotton-round-neck-t-shirt"
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters (except hyphens)
	var result strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}

	// Remove consecutive hyphens
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}
