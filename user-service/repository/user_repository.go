package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	userv1 "github.com/zabilal/microservices/pkg/genproto/user/v1"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, pageSize int32, pageToken string) ([]*User, string, error)
}

type User struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Password  string
	Status    userv1.UserStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(config DatabaseConfig) (UserRepository, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
	)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &userRepository{db: db}, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *User) error {
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
	
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) GetUser(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, first_name, last_name, password, status, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	
	user := &User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET email = ?, first_name = ?, last_name = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	
	result, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.FirstName,
		user.LastName,
		user.Status,
		time.Now(),
		user.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	query := `
		DELETE FROM users
		WHERE id = ?
	`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *userRepository) ListUsers(ctx context.Context, pageSize int32, pageToken string) ([]*User, string, error) {
	query := `
		SELECT id, email, first_name, last_name, password, status, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	offset := int32(0)
	if pageToken != "" {
		offset = pageSize * int32(len(pageToken))
	}

	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Password,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	var nextPageToken string
	if len(users) == int(pageSize) {
		nextPageToken = uuid.New().String()
	}

	return users, nextPageToken, nil
}
