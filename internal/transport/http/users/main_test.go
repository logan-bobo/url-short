package users

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"

	"url-short/internal/api"
	"url-short/internal/database"
	"url-short/internal/test/fixtures"
	"url-short/internal/test/helpers"
	testHelpers "url-short/internal/test/helpers"
	httpHelpers "url-short/internal/transport/http/helper"
)

var (
	migrations = "../../../../sql/schema/"
)

func setupUserOne(apiCfg *api.APIConfig) (CreateUserHTTPResponseBody, error) {
	request, err := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(fixtures.UserOne))

	if err != nil {
		return CreateUserHTTPResponseBody{}, err
	}

	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	userHandler := NewUserHandler(apiCfg)
	userHandler.CreateUser(response, request)

	got := CreateUserHTTPResponseBody{}

	err = json.NewDecoder(response.Body).Decode(&got)

	if err != nil {
		return CreateUserHTTPResponseBody{}, err
	}

	return got, nil
}

func loginUserOne(apiCfg *api.APIConfig) (LoginUserHTTPResponseBody, error) {
	loginRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(fixtures.UserOne))
	loginRequest.Header.Set("Content-Type", "application/json")

	loginResponse := httptest.NewRecorder()

	userHandler := NewUserHandler(apiCfg)
	userHandler.LoginUser(loginResponse, loginRequest)

	loginGot := LoginUserHTTPResponseBody{}

	err := json.NewDecoder(loginResponse.Body).Decode(&loginGot)

	if err != nil {
		return LoginUserHTTPResponseBody{}, err
	}

	return loginGot, nil
}

func TestPostUser(t *testing.T) {
	databaseId, err := helpers.NewTestDB()

	if err != nil {
		t.Errorf("could not build test database %q", err)
	}

	dbURL := os.Getenv("PG_CONN")
	testdbURL := strings.Replace(dbURL, "/postgres", fmt.Sprintf("/%s", databaseId.String()), 1)

	db, err := sql.Open("postgres", testdbURL)

	if err != nil {
		t.Errorf("can not open database connection")
	}

	defer db.Close()

	err = testHelpers.ResetDB(db, migrations)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	userAPICfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := NewUserHandler(&userAPICfg)

	t.Run("test user creation passes with correct parameters", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(fixtures.UserOne))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := CreateUserHTTPResponseBody{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		if got.Email != "test@mail.com" {
			t.Errorf("unexpected email in response, got %q, wanted %q", got.Email, "test@mail.com")
		}
	})

	t.Run("test user creation with bad parameters", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(fixtures.UserBadInput))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := httpHelpers.ErrorHTTPResponseBody{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		want := "could not create user: empty email"
		if got.Error != want {
			t.Errorf("incorrect error when invalid json used got %q wanted %q", got.Error, want)
		}
	})

	t.Run("test user creation with no body", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer([]byte(``)))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := httpHelpers.ErrorHTTPResponseBody{}

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(fixtures.UserBadEmail))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := httpHelpers.ErrorHTTPResponseBody{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		want := "could not create user: invalid email"
		if got.Error != want {
			t.Errorf("incorrect error when passing invalid email address %q wanted %q", got.Error, want)
		}
	})

	t.Run("test a duplicate user can not be created", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(fixtures.UserOne))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := httpHelpers.ErrorHTTPResponseBody{}
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

	err = testHelpers.ResetDB(db, migrations)

	if err != nil {
		t.Errorf("could not reset DB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := NewUserHandler(&apiCfg)

	_, err = setupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	t.Run("test user login fails with incorrect payload", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(fixtures.UserBadInput))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := httpHelpers.ErrorHTTPResponseBody{}

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(fixtures.UserBadEmail))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := httpHelpers.ErrorHTTPResponseBody{}

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(fixtures.UserOneBadPassword))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := httpHelpers.ErrorHTTPResponseBody{}

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(fixtures.UserOne))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := LoginUserHTTPResponseBody{}

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

	err = testHelpers.ResetDB(db, migrations)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := NewUserHandler(&apiCfg)

	_, err = setupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	t.Run("test valid user can get a new access token based on a valid refresh token", func(t *testing.T) {
		refreshRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/refresh", http.NoBody)

		buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
		refreshRequest.Header.Set("Authorization", buildHeader)

		refreshResponse := httptest.NewRecorder()

		userHandler.RefreshAccessToken(refreshResponse, refreshRequest)

		refreshGot := RefreshAccessTokenHTTPResponseBody{}

		err = json.NewDecoder(refreshResponse.Body).Decode(&refreshGot)

		if err != nil {
			t.Error("could not decode refreshResponse")
		}

		if refreshGot.AccessToken == "" {
			t.Errorf("no token was returned from refresh endpoint got %q", refreshGot.AccessToken)
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

	err = testHelpers.ResetDB(db, migrations)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := NewUserHandler(&apiCfg)

	_, err = setupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	t.Run("test user can be updated via the put user endpoint", func(t *testing.T) {
		putUserRequest, _ := http.NewRequest(http.MethodPut, "/api/v1/users", bytes.NewBuffer(fixtures.UserOneUpdatedPassword))

		buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
		putUserRequest.Header.Set("Authorization", buildHeader)

		putUserResponse := httptest.NewRecorder()

		user, err := dbQueries.SelectUser(putUserRequest.Context(), userOne.Email)

		if err != nil {
			t.Error("could not find user that was expected to exist")
		}

		userHandler.UpdateUser(putUserResponse, putUserRequest, user)

		gotPUTUser := UpdateUserHTTPResponseBody{}

		err = json.NewDecoder(putUserResponse.Body).Decode(&gotPUTUser)

		if err != nil {
			t.Error("coult not parse response")
		}

		if gotPUTUser.Email == "" || gotPUTUser.ID == 0 {
			t.Errorf("did not get expected email and ID on post user request")
		}

		userPostUpdate, err := dbQueries.SelectUser(putUserRequest.Context(), userOne.Email)

		if err != nil {
			t.Error("could not get user post password change")
		}

		err = bcrypt.CompareHashAndPassword([]byte(userPostUpdate.Password), []byte("new-password"))

		if err != nil {
			t.Errorf("hashed password did not match new password got error %q", err)
		}
	})
}
