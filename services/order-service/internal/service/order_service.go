package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/grpc/client"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/models"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/ports"
	"golang.org/x/sync/errgroup"
)

type OrderService struct {
	orderStore      ports.OrderRepository
	userClient      *client.UserClient
	productClient   *client.ProductClient
	inventoryClient *client.InventoryClient
}

func NewOrderService(orderStore ports.OrderRepository, userClient *client.UserClient, productClient *client.ProductClient,
	inventoryClient *client.InventoryClient) *OrderService {
	return &OrderService{
		orderStore:      orderStore,
		userClient:      userClient,
		productClient:   productClient,
		inventoryClient: inventoryClient,
	}
}

// CreateOrder orchestrates the complete order creation flow:
//  1. Validate user (concurrent with step 2)
//  2. Fetch + validate all products (concurrent with step 1)
//  3. Reserve inventory for all items
//  4. If any reservation fails, release already-reserved items (compensating transaction)
//  5. Create order record in database
func (s *OrderService) CreateOrder(ctx context.Context, userID string, req *models.CreateOrderRequest) (*models.Order, error) {
	// ─── Step 1 & 2: Validate user AND fetch products CONCURRENTLY ───
	g, gCtx := errgroup.WithContext(ctx)

	// Validate user
	g.Go(func() error {
		_, err := s.userClient.ValidateUser(gCtx, userID)
		if err != nil {
			return fmt.Errorf("user validation failed: %w", err)
		}
		return nil
	})

	// Fetch all products concurrently (nested errgroup)
	productDetails := make([]*models.OrderItem, len(req.Items))
	g.Go(func() error {
		pg, pCtx := errgroup.WithContext(gCtx)

		for i, item := range req.Items {
			i, item := i, item // capture loop variables
			pg.Go(func() error {
				product, err := s.productClient.GetProduct(pCtx, item.ProductID)
				if err != nil {
					return fmt.Errorf("product %s validation failed: %w", item.ProductID, err)
				}

				// Find the variant matching requested SKU
				var unitPrice float64 = product.BasePrice
				for _, v := range product.Variants {
					if v.Sku == item.SKU {
						unitPrice = v.Price
						break
					}
				}

				productDetails[i] = &models.OrderItem{
					ProductID:   item.ProductID,
					ProductName: product.Name,
					SKU:         item.SKU,
					Quantity:    item.Quantity,
					UnitPrice:   unitPrice,
					Subtotal:    unitPrice * float64(item.Quantity),
				}
				return nil
			})
		}

		return pg.Wait()
	})

	// Wait for BOTH user validation AND product fetching to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// ─── Step 3: Reserve inventory for ALL items ───
	// Track successfully reserved items for compensation if needed
	orderNumber := generateOrderNumber()
	var reservedItems []*models.OrderItem

	for _, item := range productDetails {
		err := s.inventoryClient.ReserveStock(ctx, orderNumber, item.ProductID, item.Quantity)
		if err != nil {
			// ─── Step 4: COMPENSATING TRANSACTION ───
			// Reservation failed → release everything we already reserved
			s.releaseReservedItems(ctx, orderNumber, reservedItems)
			return nil, fmt.Errorf("inventory reservation failed for product %s: %w", item.ProductID, err)
		}
		reservedItems = append(reservedItems, item)
	}

	// ─── Step 5: Calculate total and create order ───
	var totalAmount float64
	for _, item := range productDetails {
		totalAmount += item.Subtotal
	}

	order := &models.Order{
		OrderNumber: orderNumber,
		UserID:      userID,
		Status:      models.StatusPending,
		Items:       toOrderItems(productDetails),
		TotalAmount: totalAmount,
		Currency:    "INR",
	}

	createdOrder, err := s.orderStore.Create(ctx, order)
	if err != nil {
		// Order creation failed AFTER inventory was reserved
		// Must release inventory (compensating transaction)
		s.releaseReservedItems(ctx, orderNumber, reservedItems)
		return nil, fmt.Errorf("error creating order: %w", err)
	}

	return createdOrder, nil
}

// releaseReservedItems releases inventory for items that were reserved
// but the overall order creation failed. This is a COMPENSATING TRANSACTION.
func (s *OrderService) releaseReservedItems(ctx context.Context, orderNumber string, items []*models.OrderItem) {
	for _, item := range items {
		// Best effort release — log errors but don't fail further
		if err := s.inventoryClient.ReleaseStock(ctx, orderNumber, item.ProductID, item.Quantity); err != nil {
			fmt.Printf("CRITICAL: failed to release stock for order %s, product %s: %v\n",
				orderNumber, item.ProductID, err)
			// In production: this would trigger an alert for manual intervention
		}
	}
}

// GetOrder fetches an order by ID.
func (s *OrderService) GetOrder(ctx context.Context, id int64) (*models.Order, error) {
	return s.orderStore.GetByID(ctx, id)
}

// ListUserOrders fetches all orders for a user.
func (s *OrderService) ListUserOrders(ctx context.Context, userID string, limit, offset int64) ([]models.Order, error) {
	return s.orderStore.ListByUserID(ctx, userID, limit, offset)
}

// generateOrderNumber creates a human-readable order number.
func generateOrderNumber() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}

func toOrderItems(items []*models.OrderItem) []models.OrderItem {
	result := make([]models.OrderItem, len(items))
	for i, item := range items {
		result[i] = *item
	}
	return result
}
