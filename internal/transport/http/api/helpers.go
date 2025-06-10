package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"url-short/internal/domain/shorturl"
	"url-short/internal/domain/user"
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

	// json validation errros that I do not controll but must parse
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError
	if errors.As(err, &syntaxError) ||
		errors.As(err, &unmarshalTypeError) ||
		errors.As(err, &invalidUnmarshalError) {
		code = http.StatusBadRequest
	}

	switch err {
	case shorturl.ErrURLValidation:
		code = http.StatusBadRequest
	case shorturl.ErrURLNotFound:
		code = http.StatusNotFound
	case shorturl.ErrUnexpectedError:
		code = http.StatusInternalServerError
	default:
		code = http.StatusInternalServerError
	}

	switch err {
	case user.ErrEmptyEmail,
		user.ErrInvalidEmail,
		user.ErrEmptyPassword,
		user.ErrInvalidPassword,
		user.ErrInvalidLoginRequest:
		code = http.StatusBadRequest
	case user.ErrUserNotFound:
		code = http.StatusNotFound
	case user.ErrUnexpectedError:
		code = http.StatusInternalServerError
	}

	respondWithJSON(w, code, errorResponse)
}
