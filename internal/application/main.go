package application

import (
	"database/sql"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/redis/go-redis/v9"

	"url-short/internal/configuration"
	"url-short/internal/database"
	"url-short/internal/repository"
	"url-short/internal/service"
	"url-short/internal/transport/http/api"
)

type Application struct {
	Server    *http.Server
	DB        *database.Queries
	Cache     *redis.Client
	JWTSecret string
}

func NewApplication(s *configuration.ApplicationSettings) (*Application, error) {
	db, err := sql.Open("postgres", s.Database.GetPostgresDSN())
	if err != nil {
		return nil, err
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":" + s.Server.Port,
		Handler: mux,
	}

	opt, err := redis.ParseURL(s.Cache.GetCacheURL())
	if err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(opt)

	a := &Application{
		Server:    server,
		DB:        dbQueries,
		Cache:     redisClient,
		JWTSecret: s.Server.JwtSecret,
	}

	databaseRepo := repository.NewPostgresURLRepository(dbQueries)
	cacheRepo := repository.NewCacheRedis(redisClient)

	service := service.NewURLServiceImpl(databaseRepo, cacheRepo)

	users := api.NewUserHandler(a.DB, a.JWTSecret)
	auth := api.NewAuthHandler(a.DB, a.JWTSecret)
	urls := api.NewShortUrlHandler(a.DB, a.Cache, service)

	// url management endpoints
	mux.HandleFunc(
		"POST /api/v1/data/shorten",
		auth.AuthenticationMiddleware(urls.CreateShortURL),
	)
	mux.HandleFunc(
		"GET /api/v1/{shortUrl}",
		urls.GetShortURL,
	)
	mux.HandleFunc(
		"DELETE /api/v1/{shortUrl}",
		auth.AuthenticationMiddleware(urls.DeleteShortURL),
	)
	mux.HandleFunc(
		"PUT /api/v1/{shortUrl}",
		auth.AuthenticationMiddleware(urls.UpdateShortURL),
	)

	// user management endpoints
	mux.HandleFunc(
		"POST /api/v1/users",
		users.CreateUser,
	)
	mux.HandleFunc(
		"PUT /api/v1/users",
		auth.AuthenticationMiddleware(users.UpdateUser),
	)
	mux.HandleFunc(
		"POST /api/v1/login",
		users.LoginUser,
	)
	mux.HandleFunc(
		"POST /api/v1/refresh",
		users.RefreshAccessToken,
	)

	return a, nil
}
