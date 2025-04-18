package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"url-short/internal/api"
	"url-short/internal/database"
	"url-short/internal/transport/http/auth"
	"url-short/internal/transport/http/health"
	"url-short/internal/transport/http/shorturls"
	"url-short/internal/transport/http/users"
)

func main() {
	serverPort := os.Getenv("SERVER_PORT")
	dbURL := os.Getenv("PG_CONN")
	rdbURL := os.Getenv("RDB_CONN")
	jwtSecret := os.Getenv("JWT_SECRET")

	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: mux,
	}

	opt, err := redis.ParseURL(rdbURL)

	if err != nil {
		log.Fatal(err)
	}

	redisClient := redis.NewClient(opt)

	apiCfg := api.NewAPIConfig(dbQueries, redisClient, jwtSecret)

	users := users.NewUserHandler(apiCfg)
	auth := auth.NewAuthHandler(apiCfg)
	urls := shorturls.NewShortUrlHandler(apiCfg)

	// utility endpoints
	mux.HandleFunc("GET /api/v1/healthz", health.GetHealth)

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

	log.Printf("Serving port : %v \n", serverPort)
	log.Fatal(server.ListenAndServe())
}
