package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"

	"url-short/internal/database"
)

var (
	userOne                = []byte(`{"email": "test@mail.com", "password": "test"}`)
	userOneUpdatedPassword = []byte(`{"email": "test@mail.com", "password":"new-password"}`)
	userOneBadPassword     = []byte(`{"email": "test@mail.com", "password": "testerrrrr"}`)
	userBadInput           = []byte(`{"gmail":"test@mail.com", "auth": "test", "extra_data": "data"}`)
	userBadEmail           = []byte(`{"email": "test1mail.com", "password": "test"}`)
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

func setupUserOne(apiCfg *apiConfig) (*bytes.Buffer, error) {
	request, err := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userOne))

	if err != nil {
		return new(bytes.Buffer), err
	}

	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	apiCfg.postAPIUsers(response, request)

	return response.Body, err
}

func TestHealthEndpoint(t *testing.T) {
	t.Run("test healthz endpoint", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
		response := httptest.NewRecorder()

		apiCfg := apiConfig{}

		apiCfg.healthz(response, request)

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

	defer db.Close()

	err = resetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := apiConfig{
			DB: dbQueries,
	}

	t.Run("test user creation passes with correct parameters", func(t *testing.T) {
		response, err := setupUserOne(&apiCfg) 

		got := APIUsersResponse{}
		err = json.NewDecoder(response).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response, got)
		}

		if got.Email != "test@mail.com" {
			t.Errorf("unexpected email in response, got %q, wanted %q", got.Email, "test@mail.com")
		}
	})

	t.Run("test user creation with bad parameters", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userBadInput))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPIUsers(response, request)

		got := errorResponse{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		want := "incorrect parameters for user creation"
		if got.Error != want {
			t.Errorf("incorrect error when invalid json used got %q wanted %q", got.Error, want)
		}
	})

	t.Run("test user creation with no body", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer([]byte(``)))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPIUsers(response, request)

		got := errorResponse{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		want := "could not parse request"
		if got.Error != want {
			t.Errorf("incorrect error when invalid json used got %q wanted %q", got.Error, want)
		}

	})

	t.Run("test user creation with bad email address", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userBadEmail))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPIUsers(response, request)

		got := errorResponse{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		want := "invalid email address"
		if got.Error != want {
			t.Errorf("incorrect error when passing invalid email address %q wanted %q", got.Error, want)
		}
	})

	t.Run("test a duplicate user can not be created", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userOne))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPIUsers(response, request)

		got := errorResponse{}
		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		want := "could not create user in database"
		if got.Error != want {
			t.Errorf("expected duplicate user to fail got %q wanted %q", got.Error, want)
		}

	})
}

func TestPostLogin(t *testing.T) {
	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		t.Errorf("can not open database connection")
	}

	defer db.Close()

	err = resetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	// setup a user to user for this test case
	request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userOne))
	request.Header.Set("Content-Type", "application/json")

	apiCfg := apiConfig{
		DB: dbQueries,
	}

	response := httptest.NewRecorder()

	apiCfg.postAPIUsers(response, request)

	t.Run("test user login fails with incorrect payload", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(userBadInput))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPILogin(response, request)

		got := errorResponse{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("could not parse response %q", err)
		}

		want := "invalid parameters for user login"
		if got.Error != want {
			t.Errorf("incorrect error when passing invalid login parameters got %q want %q", got.Error, want)
		}
	})

	t.Run("test user login fails when user can not be found", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(userBadEmail))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPILogin(response, request)

		got := errorResponse{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("could not parse response %q", err)
		}

		want := "could not find user"
		if got.Error != want {
			t.Errorf("incorrect error when non existent user attempts to login got %q want %q", got.Error, want)
		}
	})

	t.Run("test user login fails with invalid password", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(userOneBadPassword))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPILogin(response, request)

		got := errorResponse{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("could not parse response %q", err)
		}

		want := "invalid password"
		if got.Error != want {
			t.Errorf("incorrect error when incorrect password is supplied got %q want %q", got.Error, want)
		}
	})

	t.Run("test user is returned correct ID, Email, Token and a Refresh Token", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(userOne))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		apiCfg.postAPILogin(response, request)

		got := APIUsersResponse{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("could not parse response %q", err)
		}

		if got.ID != 1 || got.Email != "test@mail.com" || got.Token == "" || got.RefreshToken == "" {
			t.Errorf("user login does not return expected results got %q", got)
		}
	})
}

func TestRefreshEndpoint(t *testing.T) {
	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		t.Errorf("can not open database connection")
	}
	
	defer db.Close()

	err = resetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	// setup a user
	request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userOne))
	request.Header.Set("Content-Type", "application/json")

	apiCfg := apiConfig{
		DB: dbQueries,
	}

	response := httptest.NewRecorder()

	apiCfg.postAPIUsers(response, request)

	t.Run("test valid user can get a new access token based on a valid refresh token", func(t *testing.T) {

		// make a request to the login endpoint to be given our token data, refresh and access
		loginRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(userOne))
		loginRequest.Header.Set("Content-Type", "application/json")

		loginResponse := httptest.NewRecorder()

		apiCfg.postAPILogin(loginResponse, loginRequest)

		loginGot := APIUsersResponse{}

		err := json.NewDecoder(loginResponse.Body).Decode(&loginGot)

		if err != nil {
			t.Errorf("could not parse login request")
		}

		// use our refresh token to be given a new access token
		refreshRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/login", http.NoBody)

		buildHeader := fmt.Sprintf("Bearer %s", loginGot.RefreshToken)
		refreshRequest.Header.Set("Authorization", buildHeader)

		refreshResponse := httptest.NewRecorder()

		apiCfg.postAPIRefresh(refreshResponse, refreshRequest)

		refreshGot := APIUsersRefreshResponse{}

		err = json.NewDecoder(refreshResponse.Body).Decode(&refreshGot)

		if err != nil {
			t.Error("could not decode refreshResponse")
		}

		if refreshGot.Token == "" {
			t.Errorf("no token was returned from refresh endpoint got %q", refreshGot.Token)
		}
	})
}

func TestPutUser(t *testing.T) {
	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		t.Errorf("can not open database connection")
	}

	defer db.Close()

	err = resetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	// setup a user
	createUserRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(userOne))
	createUserRequest.Header.Set("Content-Type", "application/json")

	apiCfg := apiConfig{
		DB: dbQueries,
	}

	createUserResponse := httptest.NewRecorder()

	apiCfg.postAPIUsers(createUserResponse, createUserRequest)

	// login user endpoint
	loginRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(userOne))
	loginRequest.Header.Set("Content-Type", "application/json")

	loginResponse := httptest.NewRecorder()

	apiCfg.postAPILogin(loginResponse, loginRequest)

	loginGot := APIUsersResponse{}

	err = json.NewDecoder(loginResponse.Body).Decode(&loginGot)

	if err != nil {
		t.Errorf("could not parse login request")
	}

	t.Run("test user can be updated via the put user endpoint", func(t *testing.T) {
		putUserRequest, _ := http.NewRequest(http.MethodPut, "/api/v1/users", bytes.NewBuffer(userOneUpdatedPassword))

		buildHeader := fmt.Sprintf("Bearer %s", loginGot.RefreshToken)
		putUserRequest.Header.Set("Authorization", buildHeader)

		putUserResponse := httptest.NewRecorder()

		user, err := dbQueries.SelectUser(putUserRequest.Context(), "test@mail.com")

		if err != nil {
			t.Error("could not find user that was expected to exist")
		}

		apiCfg.putAPIUsers(putUserResponse, putUserRequest, user)

		gotPUTUser := APIUserResponseNoToken{}

		err = json.NewDecoder(putUserResponse.Body).Decode(&gotPUTUser)

		if err != nil {
			t.Error("coult not parse response")
		}

		if gotPUTUser.Email == "" || gotPUTUser.ID == 0 {
			t.Errorf("did not get expected email and ID on post user request")
		}

		userPostUpdate, err := dbQueries.SelectUser(putUserRequest.Context(), "test@mail.com")

		if err != nil {
			t.Error("could not get user post password change")
		}

		err = bcrypt.CompareHashAndPassword([]byte(userPostUpdate.Password), []byte("new-password"))

		if err != nil {
			t.Errorf("hashed password did not match new password got error %q", err)
		}
	})
}
