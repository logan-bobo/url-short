package shorturls

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"url-short/internal/api"
	"url-short/internal/database"
	"url-short/internal/test/fixtures"
	testHelpers "url-short/internal/test/helpers"
	"url-short/internal/transport/http/users"
)

var (
	migrations = "../../../../sql/schema/"
)

func setupUserOne(apiCfg *api.APIConfig) (users.CreateUserHTTPResponseBody, error) {
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

func loginUserOne(apiCfg *api.APIConfig) (users.LoginUserHTTPResponseBody, error) {
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

func TestPostLongURL(t *testing.T) {
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

	_, err = setupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	urls := NewShortUrlHandler(&apiCfg)

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
		hash, err := HashCollisionDetection(apiCfg.DB, "www.google.com", 1, postLongURLRequest.Context())

		if err != nil {
			t.Errorf("could not generate hash err %q", err)
		}

		urls.CreateShortURL(response, postLongURLRequest, user)

		gotPutLongURL := UpdateShortURLHTTPResponseBody{}

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

	err = testHelpers.ResetDB(db, migrations)

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

	_, err = setupUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not set up user for test case with err %q", err)
	}

	userOne, err := loginUserOne(&apiCfg)

	if err != nil {
		t.Errorf("can not login user one for test case with err %q", err)
	}

	urls := NewShortUrlHandler(&apiCfg)

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

	gotPutLongURL := UpdateShortURLHTTPResponseBody{}

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
