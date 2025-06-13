package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"url-short/internal/database"
	"url-short/internal/domain/shorturl"

	"github.com/lib/pq"
)

type URLRepository interface {
	CreateShortURL(ctx context.Context, request shorturl.CreateURLRequest) (*shorturl.URL, error)
	GetURLByHash(ctx context.Context, hash string) (*shorturl.URL, error)
	UpdateShortURL(ctx context.Context, url shorturl.UpdateURLRequest) (*shorturl.URL, error)
	DeleteShortURL(ctx context.Context, url shorturl.DeleteURLRequest) error
}

type PostgresURLRepository struct {
	db *database.Queries
}

func NewPostgresURLRepository(db *database.Queries) *PostgresURLRepository {
	return &PostgresURLRepository{db: db}
}

func (r *PostgresURLRepository) CreateShortURL(
	ctx context.Context,
	url shorturl.CreateURLRequest,
) (*shorturl.URL, error) {
	now := time.Now()
	res, err := r.db.CreateURL(ctx, database.CreateURLParams{
		LongUrl:   url.LongURL,
		ShortUrl:  url.ShortURL,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    url.UserID,
	})

	if err != nil {
		return nil, getURLDomainErrorFromSQLError(err)
	}

	return &shorturl.URL{
		ID:        res.ID,
		ShortURL:  res.ShortUrl,
		LongURL:   res.LongUrl,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
		UserID:    res.UserID,
	}, nil
}

func (r *PostgresURLRepository) GetURLByHash(
	ctx context.Context,
	hash string,
) (*shorturl.URL, error) {
	res, err := r.db.SelectURL(ctx, hash)

	if err != nil {
		return nil, getURLDomainErrorFromSQLError(err)
	}

	return &shorturl.URL{
		ID:        res.ID,
		ShortURL:  res.ShortUrl,
		LongURL:   res.LongUrl,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
		UserID:    res.UserID,
	}, nil
}

func (r *PostgresURLRepository) DeleteShortURL(
	ctx context.Context,
	url shorturl.DeleteURLRequest,
) error {
	if err := r.db.DeleteURL(ctx, database.DeleteURLParams{
		UserID:   url.UserID,
		ShortUrl: url.ShortURL,
	}); err != nil {
		return getURLDomainErrorFromSQLError(err)
	}
	return nil
}

func (r *PostgresURLRepository) UpdateShortURL(
	ctx context.Context,
	url shorturl.UpdateURLRequest,
) (*shorturl.URL, error) {
	// TODO: update the query to use updated at and update the field in the repo
	res, err := r.db.UpdateShortURL(ctx, database.UpdateShortURLParams{
		UserID:   url.UserID,
		ShortUrl: url.ShortURL,
		LongUrl:  url.LongURL,
	})

	if err != nil {
		return nil, getURLDomainErrorFromSQLError(err)
	}

	return &shorturl.URL{
		ID:        res.ID,
		ShortURL:  res.ShortUrl,
		LongURL:   res.LongUrl,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
		UserID:    res.UserID,
	}, nil
}

func getURLDomainErrorFromSQLError(sqlError error) error {
	if errors.Is(sqlError, sql.ErrNoRows) {
		return shorturl.ErrURLNotFound
	}

	pgErr, ok := sqlError.(*pq.Error)
	if ok {
		// unique_violation
		if pgErr.Code == "23505" {
			return shorturl.ErrDuplicateURL
		}
	}

	return shorturl.ErrUnexpectedError
}
