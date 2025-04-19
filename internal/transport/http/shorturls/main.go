package shorturls

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"

	"url-short/internal/api"
	"url-short/internal/database"
	"url-short/internal/shortener"
	"url-short/internal/transport/http/helper"
)

type handler struct {
	// temporary now to allow transport, service and repository layers to be decoupled
	apiCfg *api.APIConfig
}

func NewShortUrlHandler(apiConfig *api.APIConfig) *handler {
	return &handler{
		apiCfg: apiConfig,
	}
}

type CreateShortURLHTTPRequest struct {
	LongURL string `json:"long_url"`
}

type CreateShortURLHTTPResponse struct {
	ShortURL string `json:"short_url"`
}

func (handler *handler) CreateShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	payload := CreateShortURLHTTPRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "incorrect request fromat")
		return
	}

	// TODO: Domain layer
	url, err := url.ParseRequestURI(payload.LongURL)

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusBadRequest, "could not parse request URL")
		return
	}

	// This should not really be the concern of the API layer
	shortURLHash, err := hashCollisionDetection(handler.apiCfg.DB, url.String(), 1, r.Context())

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusInternalServerError, "could not resolve hash collision")
		return
	}

	now := time.Now()
	shortenedURL, err := handler.apiCfg.DB.CreateURL(r.Context(), database.CreateURLParams{
		LongUrl:   url.String(),
		ShortUrl:  shortURLHash,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
	})

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusInternalServerError, "could not create short URL in database")
		return
	}

	helper.RespondWithJSON(w, http.StatusCreated, CreateShortURLHTTPResponse{
		ShortURL: shortenedURL.ShortUrl,
	})
}

// TODO: this should be moved to the service or repo layer when they are created
func hashCollisionDetection(DB *database.Queries, url string, count int, requestContext context.Context) (string, error) {
	hashURL := shortener.Hash(url, count)
	shortURLHash := shortener.Shorten(hashURL)

	_, err := DB.SelectURL(requestContext, shortURLHash)

	if err == sql.ErrNoRows {
		return shortURLHash, nil
	}

	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	count++

	return hashCollisionDetection(DB, url, count, requestContext)
}

func (handler *handler) GetShortURL(w http.ResponseWriter, r *http.Request) {
	query := r.PathValue("shortUrl")

	if query == "" {
		helper.RespondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	cacheVal, err := handler.apiCfg.Cache.Get(r.Context(), query).Result()

	switch {
	case err == redis.Nil:
		log.Printf("cache miss, key %s does not exists, writing to redis", query)

		row, err := handler.apiCfg.DB.SelectURL(r.Context(), query)

		if err != nil {
			log.Println(err)
			helper.RespondWithError(w, http.StatusInternalServerError, "database error")
			return
		}

		err = handler.apiCfg.Cache.Set(r.Context(), query, row.LongUrl, (time.Hour * 1)).Err()

		if err != nil {
			log.Printf("could not write to redis cache %s", err)
		}

		http.Redirect(w, r, row.LongUrl, http.StatusMovedPermanently)
		return

	case err != nil:
		log.Println(err)

		row, err := handler.apiCfg.DB.SelectURL(r.Context(), query)

		if err != nil {
			log.Println(err)
			helper.RespondWithError(w, http.StatusInternalServerError, "database error")
			return
		}

		http.Redirect(w, r, row.LongUrl, http.StatusMovedPermanently)
		return

	case cacheVal == "":
		log.Printf("key %s does not have a value", query)

		row, err := handler.apiCfg.DB.SelectURL(r.Context(), query)

		if err != nil {
			log.Println(err)
			helper.RespondWithError(w, http.StatusInternalServerError, "database error")
			return
		}

		http.Redirect(w, r, row.LongUrl, http.StatusMovedPermanently)
		return
	}

	log.Printf("cache hit for key %s", cacheVal)
	http.Redirect(w, r, cacheVal, http.StatusMovedPermanently)
}

func (handler *handler) DeleteShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		helper.RespondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	err := handler.apiCfg.DB.DeleteURL(r.Context(), database.DeleteURLParams{
		UserID:   user.ID,
		ShortUrl: query,
	})

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusBadRequest, "could not delete short url")
		return
	}

	w.WriteHeader(http.StatusOK)
}

type UpdateShortURLHTTPRequest struct {
	LongURL string `json:"long_url"`
}

type UpdateShortURLHTTPResponse struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func (handler *handler) UpdateShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		helper.RespondWithError(w, http.StatusBadGateway, "no short url supplied")
		return
	}

	payload := UpdateShortURLHTTPRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "invalid request body")
	}

	err = handler.apiCfg.DB.UpdateShortURL(r.Context(), database.UpdateShortURLParams{
		UserID:   user.ID,
		ShortUrl: query,
		LongUrl:  payload.LongURL,
	})

	if err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "could not update long url")
		return
	}

	err = handler.apiCfg.Cache.Set(r.Context(), query, payload.LongURL, (time.Hour * 1)).Err()

	if err != nil {
		log.Println(err)
	}

	helper.RespondWithJSON(w, http.StatusOK, UpdateShortURLHTTPResponse{
		LongURL:  payload.LongURL,
		ShortURL: query,
	})
}
