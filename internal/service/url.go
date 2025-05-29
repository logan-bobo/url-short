package service

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"url-short/internal/domain/shorturl"
	"url-short/internal/repository"

	// Really the service should not know about redis
	// but I need to implement custom errors before this can be removed
	"github.com/redis/go-redis/v9"
)

type URLService interface {
	CreateShortURL(ctx context.Context, request shorturl.CreateURLRequest) (*shorturl.URL, error)
	GenerateUniqueShortURL(ctx context.Context, longURL string) (string, error)
	GetLongURL(ctx context.Context, shortURL string) (*shorturl.URL, error)
	// UpdateShortURL(ctx context.Context, url shorturl.UpdateURLRequest) error
	// DeleteShortURL(ctx context.Context, url shorturl.DeleteURLRequest) error
}

type URLServiceImpl struct {
	urlRepo   repository.URLRepository
	cacheRepo repository.CacheRepository
}

func NewURLServiceImpl(r repository.URLRepository, c repository.CacheRepository) *URLServiceImpl {
	return &URLServiceImpl{
		urlRepo:   r,
		cacheRepo: c,
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
	urlHashPostfix := "Xa1"

	for {
		hashRes := md5.Sum([]byte(longURL + strings.Repeat(urlHashPostfix, count)))
		hash = hex.EncodeToString(hashRes[:])

		if len(hash) > 8 {
			hash = hash[:7]
		}

		_, err := s.urlRepo.GetURLByHash(ctx, hash)

		// TODO: Replace with my own errors
		if err == sql.ErrNoRows {
			return hash, nil
		}

		// TODO: Replace with my own errors
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}

		count++
	}
}

// TODO: before I continue we need custom domain errors becasue the service layer shold not know about
// a HTTP status!
func (s *URLServiceImpl) GetLongURL(ctx context.Context, shortURL string) (*shorturl.URL, error) {
	url, err := s.cacheRepo.GetURL(ctx, shortURL)

	switch {
	// cache miss
	case err == redis.Nil:
		row, err := s.urlRepo.GetURLByHash(ctx, shortURL)

		// this could be internal server error or not found...
		if err != nil {
			log.Println(err)
			return nil, err
		}

		err = s.cacheRepo.InsertURL(ctx, shortURL, row.LongURL, (time.Hour * 1))

		if err != nil {
			log.Printf("could not write to redis cache %s", err)
		}

		return row, nil

	// cache Error
	case err != nil:
		log.Println(err)

		row, err := s.urlRepo.GetURLByHash(ctx, shortURL)

		// this could be both internal server error or not found...
		if err != nil {
			log.Println(err)
			return nil, err
		}

		return row, nil

	// malformed cache Entry
	case url == "":
		row, err := s.urlRepo.GetURLByHash(ctx, shortURL)

		if err != nil {
			log.Println(err)
			return nil, err
		}

		return row, nil
	}

	return &shorturl.URL{LongURL: url}, nil
}
