package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestPostLongURL(t *testing.T) {
	app, err := withTestApplication()
	if err != nil {
		t.Errorf("could not create test app %q", err)
	}

	_, err = setupUserOne(app)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(app)
	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	urls := NewShortUrlHandler(app.URLService)

	t.Run("test user can create short URL based on long", func(t *testing.T) {
		postLongURLRequest := httptest.NewRequest(http.MethodPost, "/api/v1/urls", bytes.NewBuffer(LongUrl))

		buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
		postLongURLRequest.Header.Set("Authorization", buildHeader)

		response := httptest.NewRecorder()

		user, err := app.UserRepo.SelectUser(postLongURLRequest.Context(), userOne.Email)

		if err != nil {
			t.Error("could not find user that was expected to exist")
		}

		// As there are no hashes in the database there is no chance of a collision
		// the first hash we generate here and the hash in the call to the handler will
		// always be the same so we compare the two
		hash, err := app.URLService.GenerateUniqueShortURL(postLongURLRequest.Context(), "www.google.com")

		if err != nil {
			t.Errorf("could not generate hash err %q", err)
		}

		urls.CreateShortURL(response, postLongURLRequest, user)

		gotPutLongURL := createShortURLHTTPResponseBody{}

		err = json.NewDecoder(response.Body).Decode(&gotPutLongURL)

		if err != nil {
			t.Errorf("could not decode request err %q", err)
		}

		if gotPutLongURL.ShortURL == hash {
			t.Errorf("hash did not match")
		}
	})

	t.Run("test bad request is returned when user supplies invalid json", func(t *testing.T) {})

	t.Run("test unauthoriszed when user does not supply bearer token", func(t *testing.T) {})
}

func TestGetShortURL(t *testing.T) {
	app, err := withTestApplication()
	if err != nil {
		t.Errorf("could not create test app %q", err)
	}

	_, err = setupUserOne(app)
	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(app)
	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	urls := NewShortUrlHandler(app.URLService)

	postLongURLRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/data/shorten",
		bytes.NewBuffer(LongUrl),
	)

	buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
	postLongURLRequest.Header.Set("Authorization", buildHeader)

	postURLResponse := httptest.NewRecorder()

	user, err := app.UserRepo.SelectUser(postLongURLRequest.Context(), userOne.Email)
	if err != nil {
		t.Error("could not find user that was expected to exist")
	}

	urls.CreateShortURL(postURLResponse, postLongURLRequest, user)

	gotPutLongURL := createShortURLHTTPResponseBody{}

	err = json.NewDecoder(postURLResponse.Body).Decode(&gotPutLongURL)
	if err != nil {
		t.Errorf("could not parse get short URL response %v", err)
	}

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

func TestUpdateShortURL(t *testing.T) {
	app, err := withTestApplication()
	if err != nil {
		t.Errorf("could not create test app %q", err)
	}

	_, err = setupUserOne(app)
	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(app)
	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	urls := NewShortUrlHandler(app.URLService)

	postLongURLRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/urls",
		bytes.NewBuffer(LongUrl),
	)

	buildHeader := fmt.Sprintf("Bearer %s", userOne.RefreshToken)
	postLongURLRequest.Header.Set("Authorization", buildHeader)

	postURLResponse := httptest.NewRecorder()

	user, err := app.UserRepo.SelectUser(postLongURLRequest.Context(), userOne.Email)
	if err != nil {
		t.Error("could not find user that was expected to exist")
	}

	urls.CreateShortURL(postURLResponse, postLongURLRequest, user)

	gotPutLongURL := createShortURLHTTPResponseBody{}

	err = json.NewDecoder(postURLResponse.Body).Decode(&gotPutLongURL)
	if err != nil {
		t.Errorf("could not parse get short URL response %v", err)
	}

	t.Run("test updated at is updated when update is called on a url", func(t *testing.T) {
		now := time.Now()
		request, _ := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/urls/%s", gotPutLongURL.ShortURL),
			bytes.NewBuffer(LongUrl),
		)
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		user, err := app.UserRepo.SelectUser(postLongURLRequest.Context(), userOne.Email)

		if err != nil {
			t.Error("could not find user that was expected to exist")
		}

		urls.UpdateShortURL(response, request, user)

		got := updateShortURLHTTPResponseBody{}

		err = json.NewDecoder(response.Body).Decode(&got)

		if err != nil {
			t.Errorf("unable to parse response %q into %q", response.Body, got)
		}

		if got.UpdatedAt.After(now) {
			t.Errorf("updated at is not being updated on UpdateShortURL endpoint got %s", got)
		}
	})
}
