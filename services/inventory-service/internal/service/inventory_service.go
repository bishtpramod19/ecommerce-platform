package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/ports"
)

// InventoryService handles all business logic for inventory management.
// Combines distributed locking (Redis) with database operations (PostgreSQL).
type InventoryService struct {
	inventoryStore ports.InventoryRepository
	lock           ports.DistributedLock
	lockTTL        time.Duration
}

// NewInventoryService creates a new InventoryService.
func NewInventoryService(inventoryStore ports.InventoryRepository, lock ports.DistributedLock, lockTTLSeconds int) *InventoryService {
	return &InventoryService{
		inventoryStore: inventoryStore,
		lock:           lock,
		lockTTL:        time.Duration(lockTTLSeconds) * time.Second,
	}
}

// GetStock returns current stock levels for a product.
func (s *InventoryService) GetStock(ctx context.Context, productID string) (*model.Inventory, error) {
	inv, err := s.inventoryStore.GetByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("error fetching stock: %w", err)
	}
	return inv, nil
}

// BulkGetStock returns stock levels for multiple products.
func (s *InventoryService) BulkGetStock(ctx context.Context, productIDs []string) ([]model.Inventory, error) {
	inventories, err := s.inventoryStore.BulkGetByProductIDs(ctx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("error fetching bulk stock: %w", err)
	}
	return inventories, nil
}

// ReserveInventory atomically reserves stock for an order.
// Uses distributed lock + database transaction for double protection.
func (s *InventoryService) ReserveInventory(ctx context.Context, req *model.ReserveRequest) error {
	// Step 1: Check idempotency
	// If this order already reserved this product → return success (already done)
	alreadyReserved, err := s.inventoryStore.IsAlreadyReserved(ctx, req.OrderID, req.ProductID)
	if err != nil {
		return fmt.Errorf("error checking idempotency: %w", err)
	}
	if alreadyReserved {
		// Idempotent: same request processed twice → same result
		return nil
	}

	// Step 2: Acquire distributed lock for this product
	// Lock key: product ID (one lock per product)
	lockKey := fmt.Sprintf("product:%s", req.ProductID)

	acquired, err := s.lock.Acquire(ctx, lockKey, s.lockTTL)
	if err != nil {
		return fmt.Errorf("error acquiring lock: %w", err)
	}
	if !acquired {
		// Lock not acquired → another request is processing this product
		// Return error so order-service can retry
		return fmt.Errorf("product %s is being processed, please retry", req.ProductID)
	}

	// Step 3: Always release lock when done
	// defer ensures lock is released even if function panics
	defer s.lock.Release(ctx, lockKey)

	// Step 4: Reserve stock in database (with PostgreSQL transaction)
	if err := s.inventoryStore.Reserve(ctx, req); err != nil {
		return fmt.Errorf("error reserving stock: %w", err)
	}

	return nil
}

// ReleaseInventory releases previously reserved stock.
// Called when order is cancelled or payment fails.
func (s *InventoryService) ReleaseInventory(ctx context.Context, req *model.ReleaseRequest) error {
	// Acquire lock before releasing
	// Prevents race condition between reserve and release
	lockKey := fmt.Sprintf("product:%s", req.ProductID)

	acquired, err := s.lock.Acquire(ctx, lockKey, s.lockTTL)
	if err != nil {
		return fmt.Errorf("error acquiring lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("product %s is being processed, please retry", req.ProductID)
	}
	defer s.lock.Release(ctx, lockKey)

	if err := s.inventoryStore.Release(ctx, req); err != nil {
		return fmt.Errorf("error releasing stock: %w", err)
	}

	return nil
}

// AddStock adds new stock for a product.
// Called when new inventory arrives in warehouse.
func (s *InventoryService) AddStock(ctx context.Context, productID string, quantity int64) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}

	if err := s.inventoryStore.UpdateStock(ctx, productID, quantity); err != nil {
		return fmt.Errorf("error adding stock: %w", err)
	}

	return nil
}
