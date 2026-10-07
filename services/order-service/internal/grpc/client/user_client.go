package client

import (
	"context"
	"fmt"
	"time"

	userpb "github.com/bishtpramod19/ecommerce-protos/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	client userpb.UserServiceClient
}

func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error connecting to user-service: %w", err)
	}

	return &UserClient{client: userpb.NewUserServiceClient(conn)}, nil
}

// ValidateUser checks if a user exists and is active.
func (c *UserClient) ValidateUser(ctx context.Context, userID string) (*userpb.GetUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("error validating user: %w", err)
	}

	if !resp.IsActive {
		return nil, fmt.Errorf("user account is not active")
	}

	return resp, nil
}
