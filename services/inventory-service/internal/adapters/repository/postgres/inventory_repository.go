package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/ports"
)

type inventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) ports.InventoryRepository {
	return &inventoryRepository{db: db}
}

// GetByProductID fetches inventory for a specific product.
func (r *inventoryRepository) GetByProductID(ctx context.Context, productID string) (*model.Inventory, error) {
	query := `
		SELECT id, product_id, total_stock, reserved, available, last_updated
		FROM inventory
		WHERE product_id = $1
	`

	inv := &model.Inventory{}
	err := r.db.QueryRowContext(ctx, query, productID).Scan(
		&inv.ID,
		&inv.ProductID,
		&inv.TotalStock,
		&inv.Reserved,
		&inv.Available,
		&inv.LastUpdated,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory not found for product: %s", productID)
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching inventory: %w", err)
	}

	return inv, nil
}

// BulkGetByProductIDs fetches inventory for multiple products in one query.
func (r *inventoryRepository) BulkGetByProductIDs(ctx context.Context, productIDs []string) ([]model.Inventory, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}

	// Build placeholders: $1, $2, $3...
	placeholders := make([]string, len(productIDs))
	args := make([]interface{}, len(productIDs))
	for i, id := range productIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, product_id, total_stock, reserved, available, last_updated
		FROM inventory
		WHERE product_id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error fetching bulk inventory: %w", err)
	}
	defer rows.Close()

	var inventories []model.Inventory
	for rows.Next() {
		var inv model.Inventory
		if err := rows.Scan(
			&inv.ID,
			&inv.ProductID,
			&inv.TotalStock,
			&inv.Reserved,
			&inv.Available,
			&inv.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("error scanning inventory row: %w", err)
		}
		inventories = append(inventories, inv)
	}

	return inventories, nil
}

// Reserve atomically reserves stock using a PostgreSQL transaction.
// This is the CORE of inventory management.
func (r *inventoryRepository) Reserve(ctx context.Context, req *model.ReserveRequest) error {
	// Use a transaction to ensure atomicity
	// Both the inventory update AND reservation record must succeed together
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable, // highest isolation level
	})
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback() // rollback if anything fails

	// Step 1: Check current stock WITH a row lock
	// FOR UPDATE locks this row until transaction commits
	// Prevents other transactions from reading stale data
	var available int64
	err = tx.QueryRowContext(ctx, `
		SELECT available
		FROM inventory
		WHERE product_id = $1
		FOR UPDATE
	`, req.ProductID).Scan(&available)
	if err == sql.ErrNoRows {
		return fmt.Errorf("inventory not found for product: %s", req.ProductID)
	}
	if err != nil {
		return fmt.Errorf("error checking stock: %w", err)
	}

	// Step 2: Verify sufficient stock
	if available < req.Quantity {
		return fmt.Errorf("insufficient stock: available=%d, requested=%d",
			available, req.Quantity)
	}

	// Step 3: Update inventory (decrement available, increment reserved)
	_, err = tx.ExecContext(ctx, `
		UPDATE inventory
		SET
			reserved     = reserved + $1,
			available    = available - $1,
			last_updated = NOW()
		WHERE product_id = $2
	`, req.Quantity, req.ProductID)
	if err != nil {
		return fmt.Errorf("error updating inventory: %w", err)
	}

	// Step 4: Create reservation record (for idempotency)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO inventory_reservations (order_id, product_id, quantity, status)
		VALUES ($1, $2, $3, 'reserved')
	`, req.OrderID, req.ProductID, req.Quantity)
	if err != nil {
		return fmt.Errorf("error creating reservation: %w", err)
	}

	// Step 5: Commit transaction
	// Only now are changes visible to other transactions
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// Release releases previously reserved stock.
func (r *inventoryRepository) Release(ctx context.Context, req *model.ReleaseRequest) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Update inventory (increment available, decrement reserved)
	_, err = tx.ExecContext(ctx, `
		UPDATE inventory
		SET
			reserved     = reserved - $1,
			available    = available + $1,
			last_updated = NOW()
		WHERE product_id = $2
	`, req.Quantity, req.ProductID)
	if err != nil {
		return fmt.Errorf("error releasing inventory: %w", err)
	}

	// Update reservation status
	_, err = tx.ExecContext(ctx, `
		UPDATE inventory_reservations
		SET status = 'released', updated_at = NOW()
		WHERE order_id = $1 AND product_id = $2
	`, req.OrderID, req.ProductID)
	if err != nil {
		return fmt.Errorf("error updating reservation status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// UpdateStock updates total stock level.
func (r *inventoryRepository) UpdateStock(ctx context.Context, productID string, quantity int64) error {
	query := `
		INSERT INTO inventory (product_id, total_stock, available)
		VALUES ($1, $2, $2)
		ON CONFLICT (product_id) DO UPDATE
		SET
			total_stock  = inventory.total_stock + $2,
			available    = inventory.available + $2,
			last_updated = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, productID, quantity)
	if err != nil {
		return fmt.Errorf("error updating stock: %w", err)
	}

	return nil
}

// IsAlreadyReserved checks if an order already has a reservation.
func (r *inventoryRepository) IsAlreadyReserved(ctx context.Context, orderID string, productID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM inventory_reservations
		WHERE order_id = $1 AND product_id = $2 AND status = 'reserved'
	`, orderID, productID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking reservation: %w", err)
	}

	return count > 0, nil
}
