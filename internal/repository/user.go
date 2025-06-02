package repository

import (
	"context"
	"time"
	"url-short/internal/database"
	"url-short/internal/domain/user"
)

type UserRepository interface {
	CreateUser(ctx context.Context, request user.CreateUserRequest) (*user.User, error)
}

type PostgresUserRepository struct {
	db *database.Queries
}

func NewPostgresUserRepository(db *database.Queries) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

func (r *PostgresUserRepository) CreateUser(ctx context.Context, request user.CreateUserRequest) (*user.User, error) {
	now := time.Now()

	res, err := r.db.CreateUser(ctx, database.CreateUserParams{
		Email:     request.Email,
		Password:  request.PasswordHash,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err != nil {
		return nil, err
	}

	return &user.User{
		Id:           res.ID,
		Email:        res.Email,
		PasswordHash: []byte(res.Password),
		CreatedAt:    res.CreatedAt,
		UpdatedAt:    res.UpdatedAt,
	}, nil
}
