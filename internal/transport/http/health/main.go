package health

import (
	"net/http"

	"url-short/internal/transport/http/helper"
)

type GetHealthHTTPResponseBody struct {
	Status string `json:"status"`
}

func GetHealth(w http.ResponseWriter, r *http.Request) {
	helper.RespondWithJSON(w, http.StatusOK, GetHealthHTTPResponseBody{
		Status: "ok",
	})
}
