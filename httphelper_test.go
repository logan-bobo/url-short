package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondWithJSON(t *testing.T) {
	t.Run("test respond with json with generic interface", func(t *testing.T) {
		response := httptest.NewRecorder()

		type FooStruct struct {
			Foo string `json:"foo"`
		}

		testStruct := FooStruct{
			Foo: "bar",
		}

		respondWithJSON(response, http.StatusOK, testStruct)

		wantStruct := FooStruct{}

		err := json.NewDecoder(response.Body).Decode(&wantStruct)

		if err != nil {
			t.Errorf("could not parse result as expected")
		}

		if testStruct != wantStruct {
			t.Errorf("failed to respond with arbitrary interface got %q want %q", testStruct, wantStruct)
		}
	})
}

func TestRespondWithError(t *testing.T) {
	t.Run("test respond with error with generic error", func(t *testing.T) {
		response := httptest.NewRecorder()

		respondWithError(response, http.StatusInternalServerError, "stuff is broken")

		wantStruct := errorResponse{}

		err := json.NewDecoder(response.Body).Decode(&wantStruct)

		if err != nil {
			t.Errorf("could not parse result as expected")
		}

		if wantStruct.Error != "stuff is broken" {
			t.Errorf("failed to respond with a generic error")
		}
	})
}
