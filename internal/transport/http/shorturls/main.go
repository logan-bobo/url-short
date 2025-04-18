package shorturls

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

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

func (handler *handler) CreateLongURL(w http.ResponseWriter, r *http.Request, user database.User) {
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
