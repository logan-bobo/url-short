package main

import (
	"net/http"

	"github.com/redis/go-redis/v9"

	"url-short/internal/database"
)

type apiConfig struct {
	DB        *database.Queries
	RDB       *redis.Client
	JWTSecret string
}

type HealthResponse struct {
	Status string `json:"status"`
}

func (apiCfg *apiConfig) healthz(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
	})
}
