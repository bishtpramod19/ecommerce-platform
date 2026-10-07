package client

import (
	"context"
	"fmt"
	"time"

	productpb "github.com/bishtpramod19/ecommerce-protos/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProductClient struct {
	client productpb.ProductServiceClient
}

func NewProductClient(addr string) (*ProductClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error connecting to product-service: %w", err)
	}

	return &ProductClient{client: productpb.NewProductServiceClient(conn)}, nil
}

// GetProduct fetches product details.
func (c *ProductClient) GetProduct(ctx context.Context, productID string) (*productpb.GetProductResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.GetProduct(ctx, &productpb.GetProductRequest{ProductId: productID})
	if err != nil {
		return nil, fmt.Errorf("error fetching product: %w", err)
	}

	if !resp.IsActive {
		return nil, fmt.Errorf("product is not available")
	}

	return resp, nil
}
