package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"url-short/internal/api"
	"url-short/internal/database"
	"url-short/internal/test/fixtures"
	testHelpers "url-short/internal/test/helpers"
	httpHelpers "url-short/internal/transport/http/helper"
	"url-short/internal/transport/http/shorturls"
	"url-short/internal/transport/http/users"
)

func TestPostUser(t *testing.T) {
	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		t.Errorf("can not open database connection")
	}

	defer db.Close()

	err = testHelpers.ResetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	userAPICfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := users.NewUserHandler(&userAPICfg)

	t.Run("test user creation passes with correct parameters", func(t *testing.T) {
		userOne, err := testHelpers.SetupUserOne(&userAPICfg)

		if err != nil {
			t.Errorf("unable to setup user one due to err %q", err)
		}

		if userOne.Email != "test@mail.com" {
			t.Errorf("unexpected email in response, got %q, wanted %q", userOne.Email, "test@mail.com")
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

	err = testHelpers.ResetDB(db)

	if err != nil {
		t.Errorf("could not reset DB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := users.NewUserHandler(&apiCfg)

	_, err = testHelpers.SetupUserOne(&apiCfg)

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

		got := users.LoginUserHTTPResponseBody{}

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

	err = testHelpers.ResetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := users.NewUserHandler(&apiCfg)

	_, err = testHelpers.SetupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := testHelpers.LoginUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	t.Run("test valid user can get a new access token based on a valid refresh token", func(t *testing.T) {
		refreshRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/refresh", http.NoBody)

		buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
		refreshRequest.Header.Set("Authorization", buildHeader)

		refreshResponse := httptest.NewRecorder()

		userHandler.RefreshAccessToken(refreshResponse, refreshRequest)

		refreshGot := users.RefreshAccessTokenHTTPResponseBody{}

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

	err = testHelpers.ResetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := api.APIConfig{
		DB: dbQueries,
	}

	userHandler := users.NewUserHandler(&apiCfg)

	_, err = testHelpers.SetupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := testHelpers.LoginUserOne(&apiCfg)

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

		gotPUTUser := users.UpdateUserHTTPResponseBody{}

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

func TestPostLongURL(t *testing.T) {
	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		t.Errorf("can not open database connection")
	}

	defer db.Close()

	err = testHelpers.ResetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	apiCfg := api.APIConfig{
		DB: dbQueries,
	}

	_, err = testHelpers.SetupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := testHelpers.LoginUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	urls := shorturls.NewShortUrlHandler(&apiCfg)

	t.Run("test user can create short URL based on long", func(t *testing.T) {
		postLongURLRequest := httptest.NewRequest(http.MethodPost, "/api/v1/data/shorten", bytes.NewBuffer(fixtures.LongUrl))

		buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
		postLongURLRequest.Header.Set("Authorization", buildHeader)

		response := httptest.NewRecorder()

		user, err := dbQueries.SelectUser(postLongURLRequest.Context(), userOne.Email)

		if err != nil {
			t.Error("could not find user that was expected to exist")
		}

		// As there are no hashes in the database there is no chance of a collision
		// the first hash we generate here and the hash in the call to the handler will
		// always be the same so we compare the two
		hash, err := shorturls.HashCollisionDetection(apiCfg.DB, "www.google.com", 1, postLongURLRequest.Context())

		if err != nil {
			t.Errorf("could not generate hash err %q", err)
		}

		urls.CreateShortURL(response, postLongURLRequest, user)

		gotPutLongURL := shorturls.UpdateShortURLHTTPResponseBody{}

		err = json.NewDecoder(response.Body).Decode(&gotPutLongURL)

		if err != nil {
			t.Errorf("could not decode request err %q", err)
		}

		if gotPutLongURL.ShortURL == hash {
			t.Errorf("hash did not match")
		}
	})
}

func TestGetShortURL(t *testing.T) {
	dbURL := os.Getenv("PG_CONN")
	db, err := sql.Open("postgres", dbURL)
	rdbURL := os.Getenv("RDB_CONN")

	if err != nil {
		t.Errorf("can not open database connection")
	}

	defer db.Close()

	err = testHelpers.ResetDB(db)

	if err != nil {
		t.Errorf("could not resetDB %q", err)
	}

	dbQueries := database.New(db)

	opt, err := redis.ParseURL(rdbURL)

	if err != nil {
		t.Errorf("oculd not parse reddis connection")
	}

	redisClient := redis.NewClient(opt)

	defer redisClient.Close()

	apiCfg := api.APIConfig{
		DB:    dbQueries,
		Cache: redisClient,
	}

	_, err = testHelpers.SetupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := testHelpers.LoginUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	urls := shorturls.NewShortUrlHandler(&apiCfg)

	postLongURLRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/data/shorten",
		bytes.NewBuffer(fixtures.LongUrl),
	)

	buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
	postLongURLRequest.Header.Set("Authorization", buildHeader)

	postURLResponse := httptest.NewRecorder()

	user, err := dbQueries.SelectUser(postLongURLRequest.Context(), userOne.Email)

	if err != nil {
		t.Error("could not find user that was expected to exist")
	}

	urls.CreateShortURL(postURLResponse, postLongURLRequest, user)

	gotPutLongURL := shorturls.UpdateShortURLHTTPResponseBody{}

	_ = json.NewDecoder(postURLResponse.Body).Decode(&gotPutLongURL)

	t.Run("test short url redirects to a long URL", func(t *testing.T) {
		getShortURLRequest := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/%s", gotPutLongURL.ShortURL),
			http.NoBody,
		)

		getShortURLRequest.SetPathValue("shortUrl", gotPutLongURL.ShortURL)

		getShortURLResponse := httptest.NewRecorder()

		urls.GetShortURL(getShortURLResponse, getShortURLRequest)

		redirectLocation := getShortURLResponse.Result().Header.Get("Location")

		if redirectLocation != "https://www.google.com" {
			t.Errorf(
				"incorrect redirect to longURL got %q wanted %q",
				redirectLocation,
				"https://www.google.com",
			)
		}
	})
}
