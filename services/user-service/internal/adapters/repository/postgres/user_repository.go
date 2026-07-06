package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/ports"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) ports.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, role, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, first_name, last_name, role, is_active, created_at, updated_at
	`

	result := &model.User{}
	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Role,
		user.IsActive,
	).Scan(
		&result.ID,
		&result.Email,
		&result.FirstName,
		&result.LastName,
		&result.Role,
		&result.IsActive,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	return result, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_active = true
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = true
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, email, first_name, last_name, role, is_active, created_at, updated_at
	`

	result := &model.User{}
	err := r.db.QueryRowContext(
		ctx,
		query,
		user.FirstName,
		user.LastName,
		user.ID,
	).Scan(
		&result.ID,
		&result.Email,
		&result.FirstName,
		&result.LastName,
		&result.Role,
		&result.IsActive,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating user: %w", err)
	}

	return result, nil
}