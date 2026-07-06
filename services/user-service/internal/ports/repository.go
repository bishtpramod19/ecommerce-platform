package ports

import (
	"context"

	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) (*model.User, error)
}