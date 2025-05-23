package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheRepository interface {
	GetURL(ctx context.Context, key string) (string, error)
	InsertURL(ctx context.Context, key string, value string, cacheTime time.Duration) error
	DeleteURL(ctx context.Context, key string) error
}

type CacheRedis struct {
	cache *redis.Client
}

func NewCacheRedis(c *redis.Client) *CacheRedis {
	return &CacheRedis{
		cache: c,
	}
}

func (c CacheRedis) GetURL(ctx context.Context, key string) (string, error) {
	result, err := c.cache.Get(ctx, key).Result()

	if err == redis.Nil {
		return "", err // We should not return the redis error here we need to use a generic repo layer error of cache key not found
	}

	if err != nil {
		return "", err
	}

	return result, nil
}

func (c CacheRedis) InsertURL(ctx context.Context, key string, value string, cacheTime time.Duration) error {
	err := c.cache.Set(ctx, key, value, cacheTime).Err()
	if err != nil {
		return err
	}
	return nil
}

func (c CacheRedis) DeleteURL(ctx context.Context, key string) error {
	err := c.cache.Del(ctx, key).Err()
	if err != nil {
		return err
	}

	return nil
}
