package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"url-short/internal/database"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
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

	apiCfg := apiConfig{
		DB:        dbQueries,
		RDB:       redisClient,
		JWTSecret: jwtSecret,
	}

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
		apiCfg.postAPIUsers,
	)
	mux.HandleFunc(
		"PUT /api/v1/users",
		apiCfg.authenticationMiddleware(apiCfg.putAPIUsers),
	)
	mux.HandleFunc(
		"POST /api/v1/login",
		apiCfg.postAPILogin,
	)
	mux.HandleFunc(
		"POST /api/v1/refresh",
		apiCfg.postAPIRefresh,
	)

	log.Printf("Serving port : %v \n", serverPort)
	log.Fatal(server.ListenAndServe())
}
