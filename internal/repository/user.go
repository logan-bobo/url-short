package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"url-short/internal/database"
	"url-short/internal/domain/user"

	"github.com/lib/pq"
)

type UserRepository interface {
	CreateUser(ctx context.Context, request user.CreateUserRequest) (*user.User, error)
	SelectUser(ctx context.Context, email string) (*user.User, error)
	SelectUserByRefreshToken(ctx context.Context, email string) (*user.User, error)
	SelectUserByID(ctx context.Context, userID int32) (*user.User, error)
	UpdateRefreshToken(ctx context.Context, refreshToken string, userID int32) error
	UpdateUser(ctx context.Context, request user.UpdateUserRequest) (*user.User, error)
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
	now := time.Now().UTC()

	res, err := r.db.CreateUser(ctx, database.CreateUserParams{
		Email:     request.Email,
		Password:  request.PasswordHash,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err != nil {
		return nil, getUserDomainErrorFromSQLError(err)
	}

	return &user.User{
		Id:           res.ID,
		Email:        res.Email,
		PasswordHash: []byte(res.Password),
		CreatedAt:    res.CreatedAt,
		UpdatedAt:    res.UpdatedAt,
	}, nil
}

func (r *PostgresUserRepository) SelectUser(ctx context.Context, email string) (*user.User, error) {
	res, err := r.db.SelectUser(ctx, email)

	if err != nil {
		return nil, getUserDomainErrorFromSQLError(err)
	}

	return &user.User{
		Id:           res.ID,
		Email:        res.Email,
		PasswordHash: []byte(res.Password),
		CreatedAt:    res.CreatedAt,
		UpdatedAt:    res.UpdatedAt,
	}, nil
}

func (r *PostgresUserRepository) SelectUserByID(ctx context.Context, id int32) (*user.User, error) {
	res, err := r.db.SelectUserByID(ctx, id)
	if err != nil {
		return nil, getUserDomainErrorFromSQLError(err)
	}

	return &user.User{
		Id:           res.ID,
		Email:        res.Email,
		PasswordHash: []byte(res.Password),
		CreatedAt:    res.CreatedAt,
		UpdatedAt:    res.UpdatedAt,
	}, nil
}

func (r *PostgresUserRepository) UpdateRefreshToken(ctx context.Context, refreshToken string, userID int32) error {
	err := r.db.UserTokenRefresh(ctx, database.UserTokenRefreshParams{
		RefreshToken:           sql.NullString{String: refreshToken, Valid: true},
		RefreshTokenRevokeDate: sql.NullTime{Time: time.Now().Add(60 * (24 * time.Hour)), Valid: true},
		ID:                     userID,
	})

	if err != nil {
		return getUserDomainErrorFromSQLError(err)
	}

	return nil
}

func (r *PostgresUserRepository) SelectUserByRefreshToken(ctx context.Context, refreshToken string) (*user.User, error) {
	res, err := r.db.SelectUserByRefreshToken(ctx, sql.NullString{String: refreshToken, Valid: true})
	if err != nil {
		return nil, getUserDomainErrorFromSQLError(err)
	}

	return &user.User{
		Id:                     res.ID,
		Email:                  res.Email,
		PasswordHash:           []byte(res.Password),
		CreatedAt:              res.CreatedAt,
		UpdatedAt:              res.UpdatedAt,
		RefreshTokenRevokeDate: res.RefreshTokenRevokeDate.Time,
	}, nil
}

func (r *PostgresUserRepository) UpdateUser(ctx context.Context, request user.UpdateUserRequest) (*user.User, error) {
	res, err := r.db.UpdateUser(ctx, database.UpdateUserParams{
		Email:     request.Email,
		Password:  request.NewPasswordHash,
		ID:        request.Id,
		UpdatedAt: time.Now().UTC(),
	})

	if err != nil {
		return nil, getUserDomainErrorFromSQLError(err)
	}

	return &user.User{
		Id:        res.ID,
		Email:     res.Email,
		UpdatedAt: res.UpdatedAt,
	}, nil

}

func getUserDomainErrorFromSQLError(sqlError error) error {
	log.Println(sqlError)

	if errors.Is(sqlError, sql.ErrNoRows) {
		return user.ErrUserNotFound
	}

	pgErr, ok := sqlError.(*pq.Error)
	if ok {
		// unique_violation
		if pgErr.Code == "23505" {
			return user.ErrDuplicateUSer
		}
	}

	return user.ErrUnexpectedError
}
