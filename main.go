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

	// old api config to support old endpoints
	// removed as part of the migration
	apiCfg := apiConfig{
		DB:        dbQueries,
		RDB:       redisClient,
		JWTSecret: jwtSecret,
	}

	// allow us to only refactor the user endpoints for now
	newAPICfg := api.NewAPIConfig(dbQueries, redisClient, jwtSecret)

	userHandler := users.NewUserHandler(newAPICfg)

	// utility endpoints
	mux.HandleFunc("GET /api/v1/healthz", apiCfg.healthz)

	// url management endpoints
	mux.HandleFunc(
		"POST /api/v1/data/shorten",
		apiCfg.authenticationMiddleware(apiCfg.postLongURL),
	)
	mux.HandleFunc(
		"GET /api/v1/{shortUrl}",
		apiCfg.getShortURL,
	)
	mux.HandleFunc(
		"DELETE /api/v1/{shortUrl}",
		apiCfg.authenticationMiddleware(apiCfg.deleteShortURL),
	)
	mux.HandleFunc(
		"PUT /api/v1/{shortUrl}",
		apiCfg.authenticationMiddleware(apiCfg.putShortURL),
	)

	// user management endpoints
	mux.HandleFunc(
		"POST /api/v1/users",
		userHandler.CreateUser,
	)
	mux.HandleFunc(
		"PUT /api/v1/users",
		apiCfg.authenticationMiddleware(userHandler.UpdateUser),
	)
	mux.HandleFunc(
		"POST /api/v1/login",
		userHandler.LoginUser,
	)
	mux.HandleFunc(
		"POST /api/v1/refresh",
		userHandler.RefreshAccessToken,
	)

	log.Printf("Serving port : %v \n", serverPort)
	log.Fatal(server.ListenAndServe())
}
