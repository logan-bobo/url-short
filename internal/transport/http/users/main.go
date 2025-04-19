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
	"url-short/internal/transport/http/auth"
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

type CreateUserHTTPRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type CreateUserHTTPResponseBody struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}

func (handler *handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	payload := &CreateUserHTTPRequestBody{}

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

	helper.RespondWithJSON(w, http.StatusCreated, CreateUserHTTPResponseBody{
		ID:    user.ID,
		Email: user.Email,
	})
}

type LoginUserHTTPRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type LoginUserHTTPResponseBody struct {
	ID           int32  `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func (handler *handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	payload := LoginUserHTTPRequestBody{}

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

	helper.RespondWithJSON(w, http.StatusFound, LoginUserHTTPResponseBody{
		ID:           user.ID,
		Email:        user.Email,
		Token:        signedToken,
		RefreshToken: refreshToken,
	})
}

type RefreshAccessTokenHTTPResponseBody struct {
	// TODO: getter!
	AccessToken string `json:"token"`
}

func (handler *handler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	requestToken, err := auth.ExtractAuthTokenFromRequest(r)

	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := handler.apiCfg.DB.SelectUserByRefreshToken(r.Context(), sql.NullString{String: requestToken, Valid: true})

	if err != nil {
		helper.RespondWithError(w, http.StatusUnauthorized, "can not refresh token no user found")
		return
	}

	if time.Now().After(user.RefreshTokenRevokeDate.Time) {
		helper.RespondWithError(w, http.StatusUnauthorized, "refresh token expired, please login again")
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

	helper.RespondWithJSON(w, http.StatusCreated, RefreshAccessTokenHTTPResponseBody{
		AccessToken: signedToken,
	})
}

type UpdateUserHTTPRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type UpdateUserHTTPResponseBody struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}

func (handler *handler) UpdateUser(w http.ResponseWriter, r *http.Request, authUser database.User) {
	payload := UpdateUserHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "incorrect parameters for user update request")
		return
	}

	domainUser, err := user.NewUser(payload.Email, payload.Password)
	if err != nil {
		log.Println(err)
		helper.RespondWithError(w, http.StatusBadRequest, err.Error())
	}
	domainUser.SetID(authUser.ID)

	err = handler.apiCfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:     domainUser.Email(),
		Password:  domainUser.PasswordHash(),
		ID:        domainUser.ID(),
		UpdatedAt: time.Now(),
	})

	if err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "could not update user in database")
	}

	helper.RespondWithJSON(w, http.StatusOK, UpdateUserHTTPResponseBody{
		Email: payload.Email,
		ID:    domainUser.ID(),
	})
}
