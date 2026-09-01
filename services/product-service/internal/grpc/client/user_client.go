package client

import (
	"context"
	"fmt"

	userpb "github.com/bishtpramod19/ecommerce-protos/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserServiceClient wraps the gRPC client for user-service.
// Used by product-service to verify admin role before creating products.
type UserServiceClient struct {
	client userpb.UserServiceClient
}

// NewUserServiceClient creates a new gRPC client connected to user-service.
func NewUserServiceClient(userServiceAddr string) (*UserServiceClient, error) {
	// Create gRPC connection to user-service
	// insecure.NewCredentials() = no TLS (we'll add mTLS in Phase 4.3)
	conn, err := grpc.NewClient(
		userServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("error connecting to user-service: %w", err)
	}

	return &UserServiceClient{
		client: userpb.NewUserServiceClient(conn),
	}, nil
}

// GetUser fetches user details from user-service via gRPC.
// Called before creating a product to verify the user is an admin.
func (c *UserServiceClient) GetUser(ctx context.Context, userID string) (*userpb.GetUserResponse, error) {
	// Add timeout to gRPC call
	// Never call downstream service without timeout!
	ctx, cancel := context.WithTimeout(ctx, 5*1e9) // 5 seconds
	defer cancel()

	resp, err := c.client.GetUser(ctx, &userpb.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching user from user-service: %w", err)
	}

	return resp, nil
}

// IsAdmin checks if a user has admin role.
// Calls user-service via gRPC to get fresh user data.
func (c *UserServiceClient) IsAdmin(ctx context.Context, userID string) (bool, error) {
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return false, err
	}

	return user.Role == "admin", nil
}
