package auth

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"url-short/internal/database"
	"url-short/internal/transport/http/helper"
)

type handler struct {
	// temporary now to allow transport, service and repository layers to be decoupled
	database  *database.Queries
	JWTSecret string
}

func NewAuthHandler(database *database.Queries, JWTSecret string) *handler {
	return &handler{
		database:  database,
		JWTSecret: JWTSecret,
	}
}

func ExtractAuthTokenFromRequest(r *http.Request) (string, error) {
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

func (handler *handler) AuthenticationMiddleware(nextHandler authedHandeler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestToken, err := ExtractAuthTokenFromRequest(r)

		if err != nil {
			helper.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		claims := jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(
			requestToken,
			&claims,
			func(token *jwt.Token) (any, error) { return []byte(handler.JWTSecret), nil },
		)

		if err != nil {
			helper.RespondWithError(w, http.StatusUnauthorized, "invalid jwt")
			return
		}

		issuer, err := token.Claims.GetIssuer()

		if err != nil {
			helper.RespondWithError(w, http.StatusUnauthorized, "invalid jwt issuer")
			return
		}

		if issuer != "url-short-auth" {
			helper.RespondWithError(w, http.StatusUnauthorized, "invalid jwt issuer")
			return
		}

		userID, err := token.Claims.GetSubject()

		if err != nil {
			helper.RespondWithError(w, http.StatusUnauthorized, "could not get subject from jwt")
			return
		}

		userIDStr, err := strconv.Atoi(userID)

		if err != nil {
			helper.RespondWithError(w, http.StatusUnauthorized, "invalid jwt subject")
		}

		user, err := handler.database.SelectUserByID(r.Context(), int32(userIDStr))

		if err != nil {
			log.Println(err)
			helper.RespondWithError(w, http.StatusUnauthorized, "could not find user in database")
		}

		nextHandler(w, r, user)
	})
}
