package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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

type LongURLRequest struct {
	LongURL string `json:"long_url"`
}

type LongURLResponse struct {
	ShortURL string `json:"short_url"`
}

type ShortURLUpdateResponse struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

type APIUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type APIUsersResponse struct {
	ID           int32  `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type APIUserResponseNoToken struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}

func (apiCfg *apiConfig) healthz(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

func (apiCfg *apiConfig) deleteShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	err := apiCfg.DB.DeleteURL(r.Context(), database.DeleteURLParams{
		UserID:   user.ID,
		ShortUrl: query,
	})

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "could not delete short url")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (apiCfg *apiConfig) putShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadGateway, "no short url supplied")
		return
	}

	payload := LongURLRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
	}

	err = apiCfg.DB.UpdateShortURL(r.Context(), database.UpdateShortURLParams{
		UserID:   user.ID,
		ShortUrl: query,
		LongUrl:  payload.LongURL,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update long url")
		return
	}

	err = apiCfg.RDB.Set(r.Context(), query, payload.LongURL, (time.Hour * 1)).Err()

	if err != nil {
		log.Println(err)
	}

	respondWithJSON(w, http.StatusOK, ShortURLUpdateResponse{
		LongURL:  payload.LongURL,
		ShortURL: query,
	})
}
