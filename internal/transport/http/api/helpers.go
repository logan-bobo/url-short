package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"url-short/internal/domain/shorturl"
)

type errorHTTPResponseBody struct {
	Error string `json:"error"`
}

func respondWithJSON(w http.ResponseWriter, status int, payload any) {
	data, err := json.Marshal(payload)

	if err != nil {
		log.Printf("can not marshal payload %v", payload)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(data)

	if err != nil {
		log.Println("could not write data to response writer")
	}
}

func respondWithError(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest

	errorResponse := errorHTTPResponseBody{
		Error: err.Error(),
	}

	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError
	if errors.As(err, &syntaxError) ||
		errors.As(err, &unmarshalTypeError) ||
		errors.As(err, &invalidUnmarshalError) {
		code = http.StatusBadRequest
	}

	if errors.As(err, shorturl.ErrURLValidation) {
		code = http.StatusBadRequest
	}

	if errors.As(err, shorturl.ErrURLNotFound) {
		code = http.StatusNotFound
	}

	respondWithJSON(w, code, errorResponse)
}
