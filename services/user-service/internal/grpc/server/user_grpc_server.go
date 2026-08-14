package server

import (
	"context"
	"fmt"

	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/ports"
	userpb "github.com/bishtpramod19/ecommerce-protos/user"
)

// UserGRPCServer implements the gRPC UserServiceServer interface.
// It receives gRPC calls from other services (product-service, order-service)
// and delegates to the repository layer to fetch user data.
type UserGRPCServer struct {
	userpb.UnimplementedUserServiceServer
	userStore ports.UserRepository
}

// NewUserGRPCServer creates a new UserGRPCServer.
func NewUserGRPCServer(repo ports.UserRepository) *UserGRPCServer {
	return &UserGRPCServer{userStore: repo}
}

// GetUser handles incoming gRPC GetUser requests.
// Called by: product-service (verify admin), order-service (verify user exists)
func (s *UserGRPCServer) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	// Validate request
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// Convert string ID to int64
	var userID int64
	if _, err := fmt.Sscanf(req.UserId, "%d", &userID); err != nil {
		return nil, fmt.Errorf("invalid user_id format: %s", req.UserId)
	}

	// Fetch user from database
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Map domain model to proto response
	return &userpb.GetUserResponse{
		Id:        fmt.Sprintf("%d", user.ID),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
		IsActive:  user.IsActive,
	}, nil
}
