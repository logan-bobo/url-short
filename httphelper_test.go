package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)


func TestRespondWithJSON(t *testing.T){
	t.Run("test respond with json with generic interface", func(t *testing.T){
		response := httptest.NewRecorder()

		type fooStruct struct {
			foo string
		}

		testStruct := fooStruct{
			foo: "bar",
		}

		fmt.Println(testStruct, response.Body)

		respondWithJSON(response, http.StatusOK, testStruct)

		want := `{"foo": "bar"}`
		if response.Body.String() != want {
			t.Errorf("failed to respond with arbitrary interface got %q want %q", response.Body.String(), want)
		}
	})
}
