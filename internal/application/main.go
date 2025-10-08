package application

import (
	"database/sql"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/redis/go-redis/v9"

	"github.com/swaggo/http-swagger"
	_ "url-short/docs"

	"url-short/internal/configuration"
	"url-short/internal/database"
	"url-short/internal/repository"
	"url-short/internal/service"
	"url-short/internal/transport/http/api"
)

//	@title			URL Short
//	@version		1.0
//	@description	A simple URL shortener designed to convert longer URLs into concise, unique keys.
//	@BasePath	/

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
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         ":" + s.Server.Port,
		Handler:      mux,
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
	userRepo := repository.NewPostgresUserRepository(dbQueries)

	URLservice := service.NewURLServiceImpl(databaseRepo, cacheRepo)
	UserService := service.NewUserServiceImpl(userRepo, a.JWTSecret)

	users := api.NewUserHandler(UserService)
	auth := api.NewAuthHandler(UserService)
	urls := api.NewShortUrlHandler(URLservice)

	mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)

	mux.HandleFunc("GET /api/v1/healthz", api.GetHealth)

	mux.HandleFunc(
		"POST /api/v1/urls",
		auth.AuthenticationMiddleware(urls.CreateShortURL),
	)
	mux.HandleFunc(
		"GET /api/v1/urls/{shortUrl}",
		urls.GetShortURL,
	)
	mux.HandleFunc(
		"DELETE /api/v1/urls/{shortUrl}",
		auth.AuthenticationMiddleware(urls.DeleteShortURL),
	)
	mux.HandleFunc(
		"PUT /api/v1/urls{shortUrl}",
		auth.AuthenticationMiddleware(urls.UpdateShortURL),
	)

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
