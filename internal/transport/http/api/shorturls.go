package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"url-short/internal/domain/shorturl"
	"url-short/internal/domain/user"
	"url-short/internal/service"
)

type shorturlHandler struct {
	urlService service.URLService
}

func NewShortUrlHandler(s service.URLService) *shorturlHandler {
	return &shorturlHandler{
		urlService: s,
	}
}

type createShortURLHTTPRequestBody struct {
	LongURL string `json:"long_url"`
}

type createShortURLHTTPResponseBody struct {
	ShortURL string `json:"short_url"`
}

func (h *shorturlHandler) CreateShortURL(w http.ResponseWriter, r *http.Request, user *user.User) {
	payload := createShortURLHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println(err)
		respondWithError(w, err)
		return
	}

	createURLRequest, err := shorturl.NewCreateURLRequest(user.Id, payload.LongURL)
	if err != nil {
		log.Println(err)
		respondWithError(w, err)
		return
	}

	createURLResponse, err := h.urlService.CreateShortURL(r.Context(), *createURLRequest)
	if err != nil {
		log.Println(err)
		respondWithError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, createShortURLHTTPResponseBody{
		ShortURL: createURLResponse.ShortURL,
	})
}

func (h *shorturlHandler) GetShortURL(w http.ResponseWriter, r *http.Request) {
	shortURLExtract := r.PathValue("shortUrl")

	shortURL, err := shorturl.NewShortURL(shortURLExtract)

	if err != nil {
		respondWithError(w, err)
		return
	}

	url, err := h.urlService.GetLongURL(r.Context(), shortURL)
	if err != nil {
		respondWithError(w, err)
		return
	}

	http.Redirect(w, r, url.LongURL, http.StatusMovedPermanently)
}

func (h *shorturlHandler) DeleteShortURL(w http.ResponseWriter, r *http.Request, user *user.User) {
	shortURLExtract := r.PathValue("shortUrl")

	shortURL, err := shorturl.NewShortURL(shortURLExtract)

	if err != nil {
		respondWithError(w, err)
		return
	}

	req := shorturl.NewDeleteURLRequest(user.Id, shortURL)

	err = h.urlService.DeleteShortURL(r.Context(), *req)

	if err != nil {
		log.Println(err)
		respondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type updateShortURLHTTPRequestBody struct {
	LongURL string `json:"long_url"`
}

type updateShortURLHTTPResponseBody struct {
	ShortURL  string    `json:"short_url"`
	LongURL   string    `json:"long_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *shorturlHandler) UpdateShortURL(w http.ResponseWriter, r *http.Request, user *user.User) {
	shortURLExtract := r.PathValue("shortUrl")

	shortURL, err := shorturl.NewShortURL(shortURLExtract)

	if err != nil {
		respondWithError(w, err)
		return
	}

	payload := updateShortURLHTTPRequestBody{}

	err = json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, err)
		return
	}

	req := shorturl.NewUpdateURLRequest(user.Id, shortURL, payload.LongURL)

	url, err := h.urlService.UpdateShortURL(r.Context(), *req)

	if err != nil {
		respondWithError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, updateShortURLHTTPResponseBody{
		LongURL:   url.LongURL,
		ShortURL:  url.ShortURL,
		CreatedAt: url.CreatedAt,
		UpdatedAt: url.UpdatedAt,
	})
}
