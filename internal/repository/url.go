package repository

import (
	"context"
	"log"
	"time"

	"url-short/internal/database"
	"url-short/internal/domain/shorturl"
)

type URLRepository interface {
	CreateShortURL(ctx context.Context, request shorturl.CreateURLRequest) (*shorturl.URL, error)
	GetURLByHash(ctx context.Context, hash string) (*shorturl.URL, error)
	// UpdateShortURL(ctx context.Context, url shorturl.UpdateURLRequest) error
	// DeleteShortURL(ctx context.Context, url shorturl.DeleteURLRequest) error
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
		return nil, err
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
		log.Println(err)
		return nil, err
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
