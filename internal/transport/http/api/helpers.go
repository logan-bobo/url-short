package api

import (
	"encoding/json"
	"log"
	"net/http"
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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	errorResponse := errorHTTPResponseBody{
		Error: msg,
	}

	respondWithJSON(w, code, errorResponse)
}
