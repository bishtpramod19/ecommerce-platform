package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/models"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/ports"
)

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) ports.OrderRepository {
	return &orderRepository{db: db}
}

// Create creates a new order with its items in a single transaction.
func (r *orderRepository) Create(ctx context.Context, order *models.Order) (*models.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert order
	err = tx.QueryRowContext(ctx, `
		INSERT INTO orders (order_number, user_id, status, total_amount, currency)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, order.OrderNumber, order.UserID, order.Status, order.TotalAmount, order.Currency,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("error creating order: %w", err)
	}

	// Insert order items
	for i := range order.Items {
		item := &order.Items[i]
		item.OrderID = order.ID

		err = tx.QueryRowContext(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, sku, quantity, unit_price, subtotal)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, item.OrderID, item.ProductID, item.ProductName, item.SKU, item.Quantity, item.UnitPrice, item.Subtotal,
		).Scan(&item.ID)
		if err != nil {
			return nil, fmt.Errorf("error creating order item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return order, nil
}

// GetByID fetches an order with all its items.
func (r *orderRepository) GetByID(ctx context.Context, id int64) (*models.Order, error) {
	order := &models.Order{}

	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_number, user_id, status, total_amount, currency, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&order.ID, &order.OrderNumber, &order.UserID, &order.Status,
		&order.TotalAmount, &order.Currency, &order.CreatedAt, &order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching order: %w", err)
	}

	items, err := r.getOrderItems(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

// ListByUserID fetches paginated orders for a user.
func (r *orderRepository) ListByUserID(ctx context.Context, userID string, limit, offset int64) ([]models.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_number, user_id, status, total_amount, currency, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error fetching orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(
			&order.ID, &order.OrderNumber, &order.UserID, &order.Status,
			&order.TotalAmount, &order.Currency, &order.CreatedAt, &order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning order: %w", err)
		}

		items, err := r.getOrderItems(ctx, order.ID)
		if err != nil {
			return nil, err
		}
		order.Items = items

		orders = append(orders, order)
	}

	return orders, nil
}

// UpdateStatus updates the order status.
func (r *orderRepository) UpdateStatus(ctx context.Context, orderID int64, status models.OrderStatus) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, orderID)
	if err != nil {
		return fmt.Errorf("error updating order status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

// getOrderItems is a helper to fetch items for an order.
func (r *orderRepository) getOrderItems(ctx context.Context, orderID int64) ([]models.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, product_id, product_name, sku, quantity, unit_price, subtotal
		FROM order_items
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("error fetching order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.ProductName,
			&item.SKU, &item.Quantity, &item.UnitPrice, &item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("error scanning order item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}
