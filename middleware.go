package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"url-short/internal/database"

	"github.com/golang-jwt/jwt/v5"
)

func extractAuthTokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return "", errors.New("no authorization header supplied")
	}

	splitAuth := strings.Split(authHeader, " ")

	if len(splitAuth) == 0 {
		return "", errors.New("empty authorization header")
	}

	if len(splitAuth) != 2 && splitAuth[0] != "Bearer" {
		return "", errors.New("invalid data in authorization header")
	}

	return splitAuth[1], nil
}

type authedHandeler func(http.ResponseWriter, *http.Request, database.User)

func (apiCfg *apiConfig) authenticationMiddleware(handler authedHandeler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestToken, err := extractAuthTokenFromRequest(r)

		if err != nil {
			respondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		claims := jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(
			requestToken,
			&claims,
			func(token *jwt.Token) (interface{}, error) { return []byte(apiCfg.JWTSecret), nil },
		)

		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "invalid jwt")
			return
		}

		issuer, err := token.Claims.GetIssuer()

		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "invalid jwt issuer")
			return
		}

		if issuer != "url-short-auth" {
			respondWithError(w, http.StatusUnauthorized, "invalid jwt issuer")
			return
		}

		userID, err := token.Claims.GetSubject()

		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "could not get subject from jwt")
			return
		}

		userIDStr, err := strconv.Atoi(userID)

		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "invalid jwt subject")
		}

		user, err := apiCfg.DB.SelectUserByID(r.Context(), int32(userIDStr))

		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusUnauthorized, "could not find user in database")
		}

		handler(w, r, user)
	})
}
