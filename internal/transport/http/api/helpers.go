package api

import (
	"encoding/json"
	"errors"
	"io"
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
	var code int

	errorResponse := errorHTTPResponseBody{
		Error: err.Error(),
	}

	// json validation errros that I do not controll but must parse
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError
	if errors.As(err, &syntaxError) ||
		errors.As(err, &unmarshalTypeError) ||
		errors.Is(err, io.EOF) ||
		errors.As(err, &invalidUnmarshalError) {
		code = http.StatusBadRequest
		errorResponse = errorHTTPResponseBody{
			Error: "could not parse request",
		}
	}

	switch err {
	// user domain errros -> HTTP errors
	case user.ErrEmptyEmail,
		user.ErrInvalidEmail,
		user.ErrEmptyPassword,
		user.ErrInvalidPassword,
		user.ErrDuplicateUSer,
		user.ErrInvalidLoginRequest:
		code = http.StatusBadRequest
	case user.ErrUserNotFound:
		code = http.StatusNotFound
	case user.ErrUnexpectedError:
		code = http.StatusInternalServerError

	// authorization errors -> HTTP errors
	case ErrUnauthorized:
		code = http.StatusUnauthorized

	// shorturl domain errors -> HTTP errors
	case shorturl.ErrURLValidation,
		shorturl.ErrDuplicateURL:
		code = http.StatusBadRequest
	case shorturl.ErrURLNotFound:
		code = http.StatusNotFound
	case shorturl.ErrUnexpectedError:
		code = http.StatusInternalServerError

	default:
		code = http.StatusInternalServerError
	}

	respondWithJSON(w, code, errorResponse)
}
