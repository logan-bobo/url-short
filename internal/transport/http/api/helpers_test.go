package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"

	"url-short/internal/configuration"
	"url-short/internal/database"
	"url-short/internal/repository"
	"url-short/internal/service"
)

var (
	UserOne                = []byte(`{"email": "test@mail.com", "password": "test"}`)
	UserOneUpdatedPassword = []byte(`{"email": "test@mail.com", "password":"new-password"}`)
	UserOneBadPassword     = []byte(`{"email": "test@mail.com", "password": "testerrrrr"}`)
	UserBadInput           = []byte(`{"gmail": "test@mail.com", "auth": "test", "extra_data": "data"}`)
	UserBadEmail           = []byte(`{"email": "test1mail.com", "password": "test"}`)

	LongUrl = []byte(`{"long_url":"https://www.google.com"}`)
)

func generateRandomAlphaString(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := range result {
		result[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(result)
}

func migrateDB(db *sql.DB) error {
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS("/opt/url-short/sql/schema"),
	)

	if err != nil {
		return err
	}

	_, err = provider.Up(context.Background())

	if err != nil {
		return err
	}

	return nil
}

func withDB() (*database.Queries, error) {
	applicationSettings, err := configuration.NewApplicationSettings()
	if err != nil {
		return nil, err
	}

	dbMain, err := sql.Open("postgres", applicationSettings.Database.GetPostgresDSN())
	if err != nil {
		return nil, err
	}

	defer dbMain.Close()

	databaseName := generateRandomAlphaString(10)

	_, err = dbMain.Exec("create database " + databaseName)
	if err != nil {
		return nil, err
	}

	applicationSettings.Database.SetDatabaseName(databaseName)

	newDSN := applicationSettings.Database.GetPostgresDSN()

	testdb, err := sql.Open("postgres", newDSN)
	if err != nil {
		return nil, err
	}

	if err = migrateDB(testdb); err != nil {
		return nil, err
	}

	return database.New(testdb), nil
}

type testApplication struct {
	DB          *database.Queries
	Cache       *redis.Client
	JWTSecret   string
	CacheRepo   repository.CacheRepository
	URLRepo     repository.URLRepository
	UserRepo    repository.UserRepository
	URLService  service.URLService
	UserService service.UserService
}

func newTestApplication(s *configuration.ApplicationSettings) (*testApplication, error) {
	db, err := sql.Open("postgres", s.Database.GetPostgresDSN())
	if err != nil {
		return nil, err
	}

	dbQueries := database.New(db)

	opt, err := redis.ParseURL(s.Cache.GetCacheURL())
	if err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(opt)

	a := &testApplication{
		DB:        dbQueries,
		Cache:     redisClient,
		JWTSecret: s.Server.JwtSecret,
	}

	return a, nil
}

func withTestApplication() (*testApplication, error) {
	db, err := withDB()
	if err != nil {
		return nil, err
	}

	settings, err := configuration.NewApplicationSettings()
	if err != nil {
		return nil, err
	}

	app, err := newTestApplication(settings)
	if err != nil {
		return nil, err
	}

	// Set the applications database to the randomly generated DB
	// name from WithDB
	app.DB = db

	app.UserRepo = repository.NewPostgresUserRepository(app.DB)
	app.URLRepo = repository.NewPostgresURLRepository(app.DB)
	app.CacheRepo = repository.NewCacheRedis(app.Cache)
	app.URLService = service.NewURLServiceImpl(app.URLRepo, app.CacheRepo)
	app.UserService = service.NewUserServiceImpl(app.UserRepo, app.JWTSecret)

	return app, nil
}

func setupUserOne(a *testApplication) (*createUserHTTPResponseBody, error) {
	request, err := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(UserOne))

	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	userHandler := NewUserHandler(a.UserService)
	userHandler.CreateUser(response, request)

	got := createUserHTTPResponseBody{}

	err = json.NewDecoder(response.Body).Decode(&got)

	if err != nil {
		return nil, err
	}

	return &got, nil
}

func loginUserOne(a *testApplication) (*loginUserHTTPResponseBody, error) {
	loginRequest, _ := http.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBuffer(UserOne))
	loginRequest.Header.Set("Content-Type", "application/json")

	loginResponse := httptest.NewRecorder()

	userHandler := NewUserHandler(a.UserService)
	userHandler.LoginUser(loginResponse, loginRequest)

	loginGot := loginUserHTTPResponseBody{}

	err := json.NewDecoder(loginResponse.Body).Decode(&loginGot)

	if err != nil {
		return nil, err
	}

	return &loginGot, nil
}

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

		wantStruct := errorHTTPResponseBody{}

		err := json.NewDecoder(response.Body).Decode(&wantStruct)

		if err != nil {
			t.Errorf("could not parse result as expected")
		}

		if wantStruct.Error != "stuff is broken" {
			t.Errorf("failed to respond with a generic error")
		}
	})
}
