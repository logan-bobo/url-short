package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	t.Run("test healthz endpoint", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		response := httptest.NewRecorder()

		GetHealth(response, request)

		got := GetHealthHTTPResponseBody{}
		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		if got.Status != "ok" {
			t.Errorf("status field must be okay on health response got %q wanted %q", got.Status, "ok")
		}

		if response.Result().StatusCode != http.StatusOK {
			t.Error("endpoint must return 200")
		}
	})
}
