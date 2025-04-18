package users

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"url-short/internal/api"
	"url-short/internal/database"
	"url-short/internal/domain/user"
	"url-short/internal/transport/http/helper"
)

type handler struct {
	// temporary now to allow transport, service and repository layers to be decoupled
	apiCfg *api.APIConfig
}

func NewUserHandler(apiConfig *api.APIConfig) *handler {
	return &handler{
		apiCfg: apiConfig,
	}
}

type createUserHTTPRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type createUserHTTPResponseBody struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}

func (handler *handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	payload := &createUserHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusBadRequest, "could not parse request")
		return
	}

	domainUser, err := user.NewUser(payload.Email, payload.Password)

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := handler.apiCfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:     domainUser.Email(),
		Password:  domainUser.PasswordHash(),
		CreatedAt: domainUser.CreatedAt(),
		UpdatedAt: domainUser.UpdatedAt(),
	})

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusInternalServerError, "could not create user in database")
	}

	helper.RespondWithJSON(w, http.StatusCreated, createUserHTTPResponseBody{
		ID:    user.ID,
		Email: user.Email,
	})
}

type loginUserHTTPRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type loginUserHTTPResponseBody struct {
	ID           int32  `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func (handler *handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	payload := loginUserHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "could not parse request")
		return
	}

	if payload.Email == "" || payload.Password == "" {
		helper.RespondWithError(w, http.StatusBadRequest, "invalid parameters for user login")
	}

	// This should all be hidden behind a service and data access layer
	user, err := handler.apiCfg.DB.SelectUser(r.Context(), payload.Email)

	if err == sql.ErrNoRows {
		helper.RespondWithError(w, http.StatusNotFound, "could not find user")
		return
	}

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password))

	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "invalid password")
		return
	}

	registeredClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "url-short-auth",
		Subject:   strconv.Itoa(int(user.ID)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, registeredClaims)

	signedToken, err := token.SignedString([]byte(handler.apiCfg.JWTSecret))

	if err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "can not create JWT")
		return
	}

	byteSlice := make([]byte, 32)
	_, err = rand.Read(byteSlice)
	refreshToken := hex.EncodeToString(byteSlice)

	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusInternalServerError, "can not generate refresh token")
		return
	}

	err = handler.apiCfg.DB.UserTokenRefresh(r.Context(), database.UserTokenRefreshParams{
		RefreshToken:           sql.NullString{String: refreshToken, Valid: true},
		RefreshTokenRevokeDate: sql.NullTime{Time: time.Now().Add(60 * (24 * time.Hour)), Valid: true},
		ID:                     user.ID,
	})

	if err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "can not update user with refresh token")
		return
	}

	helper.RespondWithJSON(w, http.StatusFound, loginUserHTTPResponseBody{
		ID:           user.ID,
		Email:        user.Email,
		Token:        signedToken,
		RefreshToken: refreshToken,
	})
}
