package client

import (
	"context"
	"fmt"
	"time"

	inventorypb "github.com/bishtpramod19/ecommerce-protos/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type InventoryClient struct {
	client inventorypb.InventoryServiceClient
}

func NewInventoryClient(addr string) (*InventoryClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error connecting to inventory-service: %w", err)
	}

	return &InventoryClient{client: inventorypb.NewInventoryServiceClient(conn)}, nil
}

// ReserveStock reserves inventory for an order item.
func (c *InventoryClient) ReserveStock(ctx context.Context, orderID, productID string, quantity int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.ReserveInventory(ctx, &inventorypb.ReserveRequest{
		OrderId:   orderID,
		ProductId: productID,
		Quantity:  quantity,
	})
	if err != nil {
		return fmt.Errorf("error reserving stock: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("stock reservation failed: %s", resp.Message)
	}

	return nil
}

// ReleaseStock releases previously reserved inventory.
// Used for compensating transactions (rollback).
func (c *InventoryClient) ReleaseStock(ctx context.Context, orderID, productID string, quantity int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.ReleaseInventory(ctx, &inventorypb.ReleaseRequest{
		OrderId:   orderID,
		ProductId: productID,
		Quantity:  quantity,
	})
	if err != nil {
		return fmt.Errorf("error releasing stock: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("stock release failed: %s", resp.Message)
	}

	return nil
}
