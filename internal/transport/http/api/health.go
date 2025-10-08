package api

import (
	"net/http"
)

type getHealthHTTPResponseBody struct {
	Status string `json:"status"`
}

// GetHealth godoc
// @Summary      Health Check
// @Description  get the health status of the server
// @Tags         health
// @Produce      json
// @Success      200  {object}  getHealthHTTPResponseBody
// @Failure      500  {object}  errorHTTPResponseBody
// @Router       /api/v1/healthz [get]
func GetHealth(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, getHealthHTTPResponseBody{
		Status: "ok",
	})
}
