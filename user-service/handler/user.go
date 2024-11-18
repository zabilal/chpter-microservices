package handler

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userv1 "github.com/zabilal/microservices/pkg/genproto/user/v1"
	"github.com/zabilal/microservices/user-service/repository"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *repository.User) error
	GetUser(ctx context.Context, id string) (*repository.User, error)
}

type User struct {
	ID        string
	Email     string
	Username  string
	Password  string
	CreatedAt string
	UpdatedAt string
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(config DatabaseConfig) UserRepository {
	// Initialize database connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
	)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	return &userRepository{db: db}
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
}

type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	repo repository.UserRepository
	log  *zap.Logger
}

func NewUserHandler(repo repository.UserRepository, log *zap.Logger) *UserHandler {
	return &UserHandler{
		repo: repo,
		log:  log,
	}
}

func (h *UserHandler) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user := &repository.User{
		ID:        uuid.New().String(),
		Email:     req.GetEmail(),
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Password:  req.GetPassword(), // Note: In production, hash the password
		Status:    userv1.UserStatus_USER_STATUS_ACTIVE,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.repo.CreateUser(ctx, user); err != nil {
		h.log.Error("failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &userv1.CreateUserResponse{
		Id: user.ID,
	}, nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := h.repo.GetUser(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		h.log.Error("failed to get user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &userv1.GetUserResponse{
		User: &userv1.User{
			Id:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	user := &repository.User{
		ID:        req.GetId(),
		Email:     req.GetEmail(),
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Status:    req.GetStatus(),
		UpdatedAt: time.Now(),
	}

	if err := h.repo.UpdateUser(ctx, user); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		h.log.Error("failed to update user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &userv1.UpdateUserResponse{}, nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	if err := h.repo.DeleteUser(ctx, req.GetId()); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		h.log.Error("failed to delete user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	return &userv1.DeleteUserResponse{}, nil
}

func (h *UserHandler) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	users, nextPageToken, err := h.repo.ListUsers(ctx, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		h.log.Error("failed to list users", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	var protoUsers []*userv1.User
	for _, user := range users {
		protoUsers = append(protoUsers, &userv1.User{
			Id:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		})
	}

	return &userv1.ListUsersResponse{
		Users:         protoUsers,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *repository.User) error {
	query := `
		INSERT INTO users (id, email, first_name, last_name, password, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.FirstName,
		user.LastName,
		user.Password,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)
	
	return err
}

func (r *userRepository) GetUser(ctx context.Context, id string) (*repository.User, error) {
	query := `
		SELECT id, email, first_name, last_name, status, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	
	user := &repository.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return user, nil
}
