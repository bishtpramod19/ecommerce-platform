package server

import (
	"context"
	"fmt"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/service"
	productpb "github.com/bishtpramod19/ecommerce-protos/product"
)

// ProductGRPCServer implements the gRPC ProductServiceServer interface.
// Called by: order-service (get product details when order is placed)
type ProductGRPCServer struct {
	productpb.UnimplementedProductServiceServer
	productService *service.ProductService
}

// NewProductGRPCServer creates a new ProductGRPCServer.
func NewProductGRPCServer(productService *service.ProductService) *ProductGRPCServer {
	return &ProductGRPCServer{productService: productService}
}

// GetProduct handles incoming gRPC GetProduct requests.
// Called by order-service when a user places an order.
func (s *ProductGRPCServer) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	if req.ProductId == "" {
		return nil, fmt.Errorf("product_id is required")
	}

	product, err := s.productService.GetProduct(ctx, req.ProductId)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// Map variants
	variants := make([]*productpb.Variant, len(product.Variants))
	for i, v := range product.Variants {
		variants[i] = &productpb.Variant{
			Sku:      v.SKU,
			Size:     v.Size,
			Color:    v.Color,
			ColorHex: v.ColorHex,
			Price:    v.Price,
			Images:   v.Images,
		}
	}

	return &productpb.GetProductResponse{
		Id:          product.ID.Hex(),
		Name:        product.Name,
		Description: product.Description,
		Brand:       product.Brand,
		Category:    product.Category,
		BasePrice:   product.BasePrice,
		Currency:    product.Currency,
		Variants:    variants,
		IsActive:    product.IsActive,
	}, nil
}

// GetProducts handles batch product fetching.
// Called by order-service when order has multiple items.
func (s *ProductGRPCServer) GetProducts(ctx context.Context, req *productpb.GetProductsRequest) (*productpb.GetProductsResponse, error) {
	if len(req.ProductIds) == 0 {
		return nil, fmt.Errorf("product_ids is required")
	}

	var products []*productpb.GetProductResponse

	// Fetch each product concurrently
	type result struct {
		product *productpb.GetProductResponse
		err     error
	}

	results := make(chan result, len(req.ProductIds))

	for _, id := range req.ProductIds {
		go func(productID string) {
			product, err := s.productService.GetProduct(ctx, productID)
			if err != nil {
				results <- result{nil, err}
				return
			}

			variants := make([]*productpb.Variant, len(product.Variants))
			for i, v := range product.Variants {
				variants[i] = &productpb.Variant{
					Sku:      v.SKU,
					Size:     v.Size,
					Color:    v.Color,
					ColorHex: v.ColorHex,
					Price:    v.Price,
					Images:   v.Images,
				}
			}

			results <- result{
				product: &productpb.GetProductResponse{
					Id:          product.ID.Hex(),
					Name:        product.Name,
					Description: product.Description,
					Brand:       product.Brand,
					Category:    product.Category,
					BasePrice:   product.BasePrice,
					Currency:    product.Currency,
					Variants:    variants,
					IsActive:    product.IsActive,
				},
				err: nil,
			}
		}(id)
	}

	// Collect results
	for range req.ProductIds {
		r := <-results
		if r.err != nil {
			return nil, fmt.Errorf("error fetching product: %w", r.err)
		}
		products = append(products, r.product)
	}

	return &productpb.GetProductsResponse{
		Products: products,
	}, nil
}
