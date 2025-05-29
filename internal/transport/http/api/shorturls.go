package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"url-short/internal/database"
	"url-short/internal/domain/shorturl"
	"url-short/internal/service"
)

type shorturlHandler struct {
	// temporary now to allow transport, service and repository layers to be decoupled
	database *database.Queries
	cache    *redis.Client
	// this will be the final and ONLY field in the hanlder when I have moved to
	// a layered architechure
	urlService service.URLService
}

func NewShortUrlHandler(d *database.Queries, c *redis.Client, s service.URLService) *shorturlHandler {
	return &shorturlHandler{
		database:   d,
		cache:      c,
		urlService: s,
	}
}

type createShortURLHTTPRequestBody struct {
	LongURL string `json:"long_url"`
}

type createShortURLHTTPResponseBody struct {
	ShortURL string `json:"short_url"`
}

func (h *shorturlHandler) CreateShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	payload := createShortURLHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "incorrect request fromat")
		return
	}

	createURLRequest, err := shorturl.NewCreateURLRequest(user.ID, payload.LongURL)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "could not parse request URL")
		return
	}

	createURLResponse, err := h.urlService.CreateShortURL(r.Context(), *createURLRequest)
	if err != nil {
		log.Println(err)
		// respond with error should be able to deal with a unique key violation, not found or intenral
		// server error
		respondWithError(w, http.StatusInternalServerError, "This could be a few errors")
	}

	respondWithJSON(w, http.StatusCreated, createShortURLHTTPResponseBody{
		ShortURL: createURLResponse.ShortURL,
	})
}

func (h *shorturlHandler) GetShortURL(w http.ResponseWriter, r *http.Request) {
	shortURL := r.PathValue("shortUrl")

	if shortURL == "" {
		respondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	url, err := h.urlService.GetLongURL(r.Context(), shortURL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "This could be a few errors")
	}

	http.Redirect(w, r, url.LongURL, http.StatusMovedPermanently)
}

func (handler *shorturlHandler) DeleteShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	err := handler.database.DeleteURL(r.Context(), database.DeleteURLParams{
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

type updateShortURLHTTPRequestBody struct {
	LongURL string `json:"long_url"`
}

type updateShortURLHTTPResponseBody struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func (handler *shorturlHandler) UpdateShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadGateway, "no short url supplied")
		return
	}

	payload := updateShortURLHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
	}

	err = handler.database.UpdateShortURL(r.Context(), database.UpdateShortURLParams{
		UserID:   user.ID,
		ShortUrl: query,
		LongUrl:  payload.LongURL,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update long url")
		return
	}

	err = handler.cache.Set(r.Context(), query, payload.LongURL, (time.Hour * 1)).Err()

	if err != nil {
		log.Println(err)
	}

	respondWithJSON(w, http.StatusOK, updateShortURLHTTPResponseBody{
		LongURL:  payload.LongURL,
		ShortURL: query,
	})
}
