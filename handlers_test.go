package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pressly/goose/v3"

	"url-short/internal/database"
)

func resetDB(db *sql.DB) error {
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS("./sql/schema/"),
	)

	if err != nil {
		return errors.New("can not create goose provider")
	}

	_, err = provider.DownTo(context.Background(), 0)

	if err != nil {
		return errors.New("can not reset database")
	}

	_, err = provider.Up(context.Background())

	if err != nil {
		return errors.New("could not run migrations")
	}

	return nil 
}

func TestHealthEndpoint(t *testing.T) {
	t.Run("test healthz endpoint", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		response := httptest.NewRecorder()

		apiCfg := apiConfig{}

		apiCfg.healthz(response, request)

		// its not a good idea to use a stuct we already define in our code
		// this could introduce a subtle bug where a test could pass because
		// we incorrectly altered this struct
		got := HealthResponse{}
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

func TestPostUser(t *testing.T) {
	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		t.Errorf("can not open database connection")
	}

	err = resetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)


	t.Run("test user creation", func(t *testing.T) {
		requestJSON := []byte(`{"email": "test@mail.com", "password": "test"}`)
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(requestJSON))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg := apiConfig{
			DB: dbQueries,
		}

		apiCfg.postAPIUsers(response, request)

		got := APIUsersResponse{}
		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		if got.Email != "test@mail.com" {
			t.Errorf("unexpected email in response, got %q, wanted %q", got.Email, "test@mail.com")
		}
	})
}
