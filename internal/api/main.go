package api

import (
	"github.com/redis/go-redis/v9"
	"url-short/internal/database"
)

type APIConfig struct {
	DB        *database.Queries
	Cache     *redis.Client
	JWTSecret string
}

func NewAPIConfig(db *database.Queries, cache *redis.Client, jwtSecret string) *APIConfig {
	return &APIConfig{
		DB:        db,
		Cache:     cache,
		JWTSecret: jwtSecret,
	}
}
