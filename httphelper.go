package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	data, err := json.Marshal(payload)

	if err != nil {
		log.Printf("can not Marshal payload %v", payload)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	errorResponse := struct {
		Error string `json:"error"`
	}{
		Error: msg,
	}

	respondWithJSON(w, code, errorResponse)
}