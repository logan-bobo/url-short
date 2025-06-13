package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"url-short/internal/domain/user"
	"url-short/internal/service"
)

type authHandler struct {
	service service.UserService
}

func NewAuthHandler(service service.UserService) *authHandler {
	return &authHandler{
		service: service,
	}
}

var (
	ErrUnauthorized = errors.New("unauthorized")
)

func ExtractAuthTokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return "", ErrUnauthorized
	}

	splitAuth := strings.Split(authHeader, " ")

	if len(splitAuth) == 0 {
		return "", ErrUnauthorized
	}

	if len(splitAuth) != 2 && splitAuth[0] != "Bearer" {
		return "", ErrUnauthorized
	}

	return splitAuth[1], nil
}

type authedHandeler func(http.ResponseWriter, *http.Request, *user.User)

func (handler *authHandler) AuthenticationMiddleware(nextHandler authedHandeler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestToken, err := ExtractAuthTokenFromRequest(r)

		if err != nil {
			respondWithError(w, err)
			return
		}

		user, err := handler.service.ValidateUserJWT(r.Context(), requestToken)
		if err != nil {
			log.Println(err)
			respondWithError(w, ErrUnauthorized)
			return
		}

		nextHandler(w, r, user)
	})
}
