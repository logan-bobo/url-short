package helpers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/pressly/goose/v3"

	"url-short/internal/api"
	"url-short/internal/test/fixtures"
	"url-short/internal/transport/http/users"
)

func ResetDB(db *sql.DB) error {
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

func SetupUserOne(apiCfg *api.APIConfig) (users.CreateUserHTTPResponseBody, error) {
	request, err := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(fixtures.UserOne))

	if err != nil {
		return users.CreateUserHTTPResponseBody{}, err
	}

	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	userHandler := users.NewUserHandler(apiCfg)
	userHandler.CreateUser(response, request)

	got := users.CreateUserHTTPResponseBody{}

	err = json.NewDecoder(response.Body).Decode(&got)

	if err != nil {
		return users.CreateUserHTTPResponseBody{}, err
	}

	return got, nil
}

func LoginUserOne(apiCfg *api.APIConfig) (users.LoginUserHTTPResponseBody, error) {
	loginRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(fixtures.UserOne))
	loginRequest.Header.Set("Content-Type", "application/json")

	loginResponse := httptest.NewRecorder()

	userHandler := users.NewUserHandler(apiCfg)
	userHandler.LoginUser(loginResponse, loginRequest)

	loginGot := users.LoginUserHTTPResponseBody{}

	err := json.NewDecoder(loginResponse.Body).Decode(&loginGot)

	if err != nil {
		return users.LoginUserHTTPResponseBody{}, err
	}

	return loginGot, nil
}
