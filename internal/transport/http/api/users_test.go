package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
)

func TestPostUser(t *testing.T) {
	app, err := withTestApplication()
	if err != nil {
		t.Errorf("could not create test app %q", err)
	}

	userHandler := NewUserHandler(app.UserService)

	t.Run("test user creation passes with correct parameters", func(t *testing.T) {
		userOne, err := setupUserOne(app)

		if err != nil {
			t.Errorf("unable to setup user one due to err %q", err)
		}

		if userOne.Email != "test@mail.com" {
			t.Errorf("unexpected email in response, got %q, wanted %q", userOne.Email, "test@mail.com")
		}
	})

	t.Run("test user creation with bad parameters", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(UserBadInput))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := errorHTTPResponseBody{}

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

		got := errorHTTPResponseBody{}

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(UserBadEmail))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := errorHTTPResponseBody{}

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(UserOne))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.CreateUser(response, request)

		got := errorHTTPResponseBody{}
		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		want := "could not create user"
		if got.Error != want {
			t.Errorf("expected duplicate user to fail got %q wanted %q", got.Error, want)
		}
	})
}

func TestPostLogin(t *testing.T) {
	app, err := withTestApplication()
	if err != nil {
		t.Errorf("could not create test app %q", err)
	}

	userHandler := NewUserHandler(app.UserService)

	_, err = setupUserOne(app)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	t.Run("test user login fails with incorrect payload", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(UserBadInput))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := errorHTTPResponseBody{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("could not parse response %q", err)
		}

		want := "email or password must not be empty"
		if got.Error != want {
			t.Errorf("incorrect error when passing invalid login parameters got %q want %q", got.Error, want)
		}
	})

	t.Run("test user login fails when user can not be found", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(UserBadEmail))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := errorHTTPResponseBody{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("could not parse response %q", err)
		}

		want := "could be lots of errors need to handle"
		if got.Error != want {
			t.Errorf("incorrect error when non existent user attempts to login got %q want %q", got.Error, want)
		}
	})

	t.Run("test user login fails with invalid password", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(UserOneBadPassword))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := errorHTTPResponseBody{}

		err := json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("could not parse response %q", err)
		}

		want := "could be lots of errors need to handle"
		if got.Error != want {
			t.Errorf("incorrect error when incorrect password is supplied got %q want %q", got.Error, want)
		}
	})

	t.Run("test user is returned correct ID, Email, Token and a Refresh Token", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(UserOne))
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		userHandler.LoginUser(response, request)

		got := loginUserHTTPResponseBody{}

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
	app, err := withTestApplication()
	if err != nil {
		t.Errorf("could not create test app %q", err)
	}

	userHandler := NewUserHandler(app.UserService)

	_, err = setupUserOne(app)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(app)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	t.Run("test valid user can get a new access token based on a valid refresh token", func(t *testing.T) {
		refreshRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/refresh", http.NoBody)

		buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
		refreshRequest.Header.Set("Authorization", buildHeader)

		refreshResponse := httptest.NewRecorder()

		userHandler.RefreshAccessToken(refreshResponse, refreshRequest)

		refreshGot := refreshAccessTokenHTTPResponseBody{}

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
	app, err := withTestApplication()
	if err != nil {
		t.Errorf("could not create test app %q", err)
	}

	userHandler := NewUserHandler(app.UserService)

	_, err = setupUserOne(app)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(app)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	t.Run("test user can be updated via the put user endpoint", func(t *testing.T) {
		putUserRequest, _ := http.NewRequest(http.MethodPut, "/api/v1/users", bytes.NewBuffer(UserOneUpdatedPassword))

		buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
		putUserRequest.Header.Set("Authorization", buildHeader)

		putUserResponse := httptest.NewRecorder()

		user, err := app.DB.SelectUser(putUserRequest.Context(), userOne.Email)

		if err != nil {
			t.Error("could not find user that was expected to exist")
		}

		userHandler.UpdateUser(putUserResponse, putUserRequest, user)

		gotPUTUser := updateUserHTTPResponseBody{}

		err = json.NewDecoder(putUserResponse.Body).Decode(&gotPUTUser)

		if err != nil {
			t.Error("coult not parse response")
		}

		if gotPUTUser.Email == "" || gotPUTUser.ID == 0 {
			t.Errorf("did not get expected email and ID on post user request")
		}

		userPostUpdate, err := app.DB.SelectUser(putUserRequest.Context(), userOne.Email)

		if err != nil {
			t.Error("could not get user post password change")
		}

		err = bcrypt.CompareHashAndPassword([]byte(userPostUpdate.Password), []byte("new-password"))

		if err != nil {
			t.Errorf("hashed password did not match new password got error %q", err)
		}
	})
}
