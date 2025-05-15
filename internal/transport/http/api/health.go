package api

import (
	"net/http"
)

type getHealthHTTPResponseBody struct {
	Status string `json:"status"`
}

func GetHealth(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, getHealthHTTPResponseBody{
		Status: "ok",
	})
}
