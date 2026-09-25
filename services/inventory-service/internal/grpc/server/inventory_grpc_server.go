package server

import (
	"context"
	"fmt"

	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/service"
	inventorypb "github.com/bishtpramod19/ecommerce-protos/inventory"
)

// InventoryGRPCServer is the gRPC InventoryServiceServer implementation
// called by : order service
type InventoryGRPCServer struct {
	inventorypb.UnimplementedInventoryServiceServer
	inventoryService *service.InventoryService
}

// NewInventoryGRPCServer creates a new InventoryGRPCServer.
func NewInventoryGRPCServer(inventoryService *service.InventoryService) *InventoryGRPCServer {
	return &InventoryGRPCServer{
		inventoryService: inventoryService,
	}
}

// GetStock handles incoming GetStock gRPC requests.
func (s *InventoryGRPCServer) GetStock(ctx context.Context, req *inventorypb.GetStockRequest) (*inventorypb.GetStockResponse, error) {
	if req.ProductId == "" {
		return nil, fmt.Errorf("product_id is required")
	}

	inv, err := s.inventoryService.GetStock(ctx, req.ProductId)
	if err != nil {
		return nil, fmt.Errorf("error fetching stock: %w", err)
	}

	return &inventorypb.GetStockResponse{
		ProductId:  inv.ProductID,
		TotalStock: inv.TotalStock,
		Reserved:   inv.Reserved,
		Available:  inv.Available,
	}, nil
}

// BulkGetStock handles batch stock queries.
func (s *InventoryGRPCServer) BulkGetStock(ctx context.Context, req *inventorypb.BulkGetStockRequest) (*inventorypb.BulkGetStockResponse, error) {
	if len(req.ProductIds) == 0 {
		return nil, fmt.Errorf("product_ids is required")
	}

	inventories, err := s.inventoryService.BulkGetStock(ctx, req.ProductIds)
	if err != nil {
		return nil, fmt.Errorf("error fetching bulk stock: %w", err)
	}

	stocks := make([]*inventorypb.GetStockResponse, len(inventories))
	for i, inv := range inventories {
		stocks[i] = &inventorypb.GetStockResponse{
			ProductId:  inv.ProductID,
			TotalStock: inv.TotalStock,
			Reserved:   inv.Reserved,
			Available:  inv.Available,
		}
	}

	return &inventorypb.BulkGetStockResponse{Stocks: stocks}, nil
}

// ReserveInventory handles stock reservation requests.
func (s *InventoryGRPCServer) ReserveInventory(ctx context.Context, req *inventorypb.ReserveRequest) (*inventorypb.ReserveResponse, error) {
	if req.ProductId == "" || req.OrderId == "" {
		return nil, fmt.Errorf("product_id and order_id are required")
	}
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	err := s.inventoryService.ReserveInventory(ctx, &model.ReserveRequest{
		ProductID: req.ProductId,
		Quantity:  req.Quantity,
		OrderID:   req.OrderId,
	})
	if err != nil {
		return &inventorypb.ReserveResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &inventorypb.ReserveResponse{
		Success: true,
		Message: "inventory reserved successfully",
	}, nil
}

// ReleaseInventory handles stock release requests.
func (s *InventoryGRPCServer) ReleaseInventory(ctx context.Context, req *inventorypb.ReleaseRequest) (*inventorypb.ReleaseResponse, error) {
	if req.ProductId == "" || req.OrderId == "" {
		return nil, fmt.Errorf("product_id and order_id are required")
	}

	err := s.inventoryService.ReleaseInventory(ctx, &model.ReleaseRequest{
		ProductID: req.ProductId,
		Quantity:  req.Quantity,
		OrderID:   req.OrderId,
	})
	if err != nil {
		return &inventorypb.ReleaseResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &inventorypb.ReleaseResponse{
		Success: true,
		Message: "inventory released successfully",
	}, nil
}
