// test/integration/user_service_test.go
package integration

import (
		"context"
		"testing"
		"time"

		"github.com/stretchr/testify/require"
		"google.golang.org/grpc"

		pb "github.com/yourusername/microservices/pkg/genproto/user/v1"
)

func TestUserService(t *testing.T) {
		// Set up connection to user service
		conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
		require.NoError(t, err)
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test user creation
		t.Run("CreateUser", func(t *testing.T) {
				resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
						Email: "test@example.com",
						Name:  "Test User",
				})
				require.NoError(t, err)
				require.NotEmpty(t, resp.User.Id)
				require.Equal(t, "test@example.com", resp.User.Email)

				// Store user ID for subsequent tests
				userID := resp.User.Id

				// Test get user
				t.Run("GetUser", func(t *testing.T) {
						resp, err := client.GetUser(ctx, &pb.GetUserRequest{
								UserId: userID,
						})
						require.NoError(t, err)
						require.Equal(t, userID, resp.User.Id)
						require.Equal(t, "test@example.com", resp.User.Email)
				})
		})
}