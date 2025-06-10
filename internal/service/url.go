package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"url-short/internal/domain/shorturl"
	"url-short/internal/repository"

	"github.com/redis/go-redis/v9"
)

type URLService interface {
	CreateShortURL(ctx context.Context, request shorturl.CreateURLRequest) (*shorturl.URL, error)
	GenerateUniqueShortURL(ctx context.Context, longURL string) (string, error)
	GetLongURL(ctx context.Context, shortURL string) (*shorturl.URL, error)
	UpdateShortURL(ctx context.Context, url shorturl.UpdateURLRequest) (*shorturl.URL, error)
	DeleteShortURL(ctx context.Context, url shorturl.DeleteURLRequest) error
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
		if err == shorturl.ErrURLNotFound {
			return hash, nil
		}
		if err != nil && err != shorturl.ErrURLNotFound {
			return "", err
		}

		count++
	}
}

func (s *URLServiceImpl) GetLongURL(ctx context.Context, shortURL string) (*shorturl.URL, error) {
	url, err := s.cacheRepo.GetURL(ctx, shortURL)

	switch {
	// cache miss
	case err == redis.Nil:
		row, err := s.urlRepo.GetURLByHash(ctx, shortURL)

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

func (s *URLServiceImpl) DeleteShortURL(ctx context.Context, url shorturl.DeleteURLRequest) error {
	if err := s.urlRepo.DeleteShortURL(ctx, url); err != nil {
		return err
	}
	return nil
}

func (s *URLServiceImpl) UpdateShortURL(ctx context.Context, request shorturl.UpdateURLRequest) (*shorturl.URL, error) {
	url, err := s.urlRepo.UpdateShortURL(ctx, request)

	if err != nil {
		return nil, err
	}

	err = s.cacheRepo.InsertURL(ctx, url.ShortURL, url.LongURL, (time.Hour * 1))

	if err != nil {
		// We can use a cache write failure metric here to avoid not returning the updated url to the user
		log.Println(err)
	}

	return url, nil
}
