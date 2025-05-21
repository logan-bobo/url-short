package service

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"log"
	"strings"

	"url-short/internal/domain/shorturl"
	"url-short/internal/repository"
)

type URLService interface {
	CreateShortURL(ctx context.Context, request shorturl.CreateURLRequest) (*shorturl.URL, error)
	GenerateUniqueShortURL(ctx context.Context, longURL string) (string, error)
	// GetLongURL(ctx context.Context, url shorturl.GetURLByShortRequest) (shorturl.URL, error)
	// UpdateShortURL(ctx context.Context, url shorturl.UpdateURLRequest) error
	// DeleteShortURL(ctx context.Context, url shorturl.DeleteURLRequest) error
}

type URLServiceImpl struct {
	urlRepo repository.URLRepository
	// cacheRepo repository.CacheRepository
}

func NewURLServiceImpl(r repository.URLRepository) *URLServiceImpl {
	return &URLServiceImpl{
		urlRepo: r,
	}
}

func (s *URLServiceImpl) CreateShortURL(
	ctx context.Context,
	request shorturl.CreateURLRequest,
) (*shorturl.URL, error) {
	shortURLHash, err := s.GenerateUniqueShortURL(ctx, request.LongURL)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	request.ShortURL = shortURLHash

	createdShortURL, err := s.urlRepo.CreateShortURL(ctx, request)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return createdShortURL, nil
}

func (s *URLServiceImpl) GenerateUniqueShortURL(
	ctx context.Context,
	longURL string,
) (string, error) {
	count := 0
	hash := ""
	foundHash := true
	urlHashPostfix := "Xa1"

	for foundHash {
		hashRes := md5.Sum([]byte(longURL + strings.Repeat(urlHashPostfix, count)))
		hash = hex.EncodeToString(hashRes[:])

		if len(hash) < 8 {
			hash = hash[:7]
		}

		_, err := s.urlRepo.GetURLByHash(ctx, hash)

		// TODO: Replace with my own errors
		if err == sql.ErrNoRows {
			foundHash = false
		}

		// TODO: Replace with my own errors
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}

		count++
	}

	return hash, nil

}
