package api

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

	"url-short/internal/database"
	"url-short/internal/domain/user"
	"url-short/internal/service"
)

type userHandler struct {
	database    *database.Queries
	JWTSecret   string
	userService service.UserService
}

func NewUserHandler(database *database.Queries, JWTSecret string, userService service.UserService) *userHandler {
	return &userHandler{
		database:    database,
		JWTSecret:   JWTSecret,
		userService: userService,
	}
}

type createUserHTTPRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createUserHTTPResponseBody struct {
	ID        int32  `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (handler *userHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	payload := &createUserHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "could not parse request")
		return
	}

	createUserRequest, err := user.NewCreateUserRequest(payload.Email, payload.Password)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := handler.userService.CreateUser(r.Context(), *createUserRequest)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	respondWithJSON(w, http.StatusCreated, createUserHTTPResponseBody{
		ID:        res.Id,
		Email:     res.Email,
		CreatedAt: res.CreatedAt.String(),
		UpdatedAt: res.UpdatedAt.String(),
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

func (handler *userHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	payload := loginUserHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not parse request")
		return
	}

	if payload.Email == "" || payload.Password == "" {
		respondWithError(w, http.StatusBadRequest, "invalid parameters for user login")
	}

	// This should all be hidden behind a service and data access layer
	user, err := handler.database.SelectUser(r.Context(), payload.Email)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "could not find user")
		return
	}

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "database error")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password))

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid password")
		return
	}

	registeredClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "url-short-auth",
		Subject:   strconv.Itoa(int(user.ID)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, registeredClaims)

	signedToken, err := token.SignedString([]byte(handler.JWTSecret))

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can not create JWT")
		return
	}

	byteSlice := make([]byte, 32)
	_, err = rand.Read(byteSlice)
	refreshToken := hex.EncodeToString(byteSlice)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "can not generate refresh token")
		return
	}

	err = handler.database.UserTokenRefresh(r.Context(), database.UserTokenRefreshParams{
		RefreshToken:           sql.NullString{String: refreshToken, Valid: true},
		RefreshTokenRevokeDate: sql.NullTime{Time: time.Now().Add(60 * (24 * time.Hour)), Valid: true},
		ID:                     user.ID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can not update user with refresh token")
		return
	}

	respondWithJSON(w, http.StatusFound, loginUserHTTPResponseBody{
		ID:           user.ID,
		Email:        user.Email,
		Token:        signedToken,
		RefreshToken: refreshToken,
	})
}

type refreshAccessTokenHTTPResponseBody struct {
	// TODO: getter!
	AccessToken string `json:"token"`
}

func (handler *userHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	requestToken, err := ExtractAuthTokenFromRequest(r)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := handler.database.SelectUserByRefreshToken(r.Context(), sql.NullString{String: requestToken, Valid: true})

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "can not refresh token no user found")
		return
	}

	if time.Now().After(user.RefreshTokenRevokeDate.Time) {
		respondWithError(w, http.StatusUnauthorized, "refresh token expired, please login again")
		return
	}

	registeredClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "url-short-auth",
		Subject:   strconv.Itoa(int(user.ID)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, registeredClaims)

	signedToken, err := token.SignedString([]byte(handler.JWTSecret))

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can not create JWT")
		return
	}

	respondWithJSON(w, http.StatusCreated, refreshAccessTokenHTTPResponseBody{
		AccessToken: signedToken,
	})
}

type updateUserHTTPRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type updateUserHTTPResponseBody struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}

func (handler *userHandler) UpdateUser(w http.ResponseWriter, r *http.Request, authUser database.User) {
	payload := updateUserHTTPRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "incorrect parameters for user update request")
		return
	}

	domainUser, err := user.NewUser(payload.Email, payload.Password)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	domainUser.Id = authUser.ID

	err = handler.database.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:     domainUser.Email,
		Password:  domainUser.GetPasswordHash(),
		ID:        domainUser.Id,
		UpdatedAt: time.Now(),
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update user in database")
	}

	respondWithJSON(w, http.StatusOK, updateUserHTTPResponseBody{
		Email: payload.Email,
		ID:    domainUser.Id,
	})
}
