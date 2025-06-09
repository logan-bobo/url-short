package api

import (
	"encoding/json"
	"log"
	"net/http"

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
		respondWithError(w, http.StatusBadRequest, "incorrect request fromat")
		return
	}

	createURLRequest, err := shorturl.NewCreateURLRequest(user.Id, payload.LongURL)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, err)
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

func (h *shorturlHandler) DeleteShortURL(w http.ResponseWriter, r *http.Request, user *user.User) {
	shortUrl := r.PathValue("shortUrl")

	if shortUrl == "" {
		respondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	req := shorturl.NewDeleteURLRequest(user.Id, shortUrl)

	err := h.urlService.DeleteShortURL(r.Context(), *req)

	if err != nil {
		log.Println(err)
		// TODO: multiple errors could bubble up here handle them!
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

func (h *shorturlHandler) UpdateShortURL(w http.ResponseWriter, r *http.Request, user *user.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	payload := updateShortURLHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
	}

	req := shorturl.NewUpdateURLRequest(user.Id, query, payload.LongURL)

	url, err := h.urlService.UpdateShortURL(r.Context(), *req)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "this could be a few errors come back to handle")
	}

	respondWithJSON(w, http.StatusOK, updateShortURLHTTPResponseBody{
		LongURL:  url.LongURL,
		ShortURL: url.ShortURL,
	})
}
