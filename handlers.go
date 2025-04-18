package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"url-short/internal/database"
	"url-short/internal/domain/user"
)

type apiConfig struct {
	DB        *database.Queries
	RDB       *redis.Client
	JWTSecret string
}

type HealthResponse struct {
	Status string `json:"status"`
}

type LongURLRequest struct {
	LongURL string `json:"long_url"`
}

type LongURLResponse struct {
	ShortURL string `json:"short_url"`
}

type ShortURLUpdateResponse struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

type APIUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type APIUsersResponse struct {
	ID           int32  `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type APIUsersRefreshResponse struct {
	Token string `json:"token"`
}

type APIUserResponseNoToken struct {
	ID    int32  `json:"id"`
	Email string `json:"email"`
}

func (apiCfg *apiConfig) healthz(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

func (apiCfg *apiConfig) postLongURL(w http.ResponseWriter, r *http.Request, user database.User) {
	payload := LongURLRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "incorrect request fromat")
		return
	}

	url, err := url.ParseRequestURI(payload.LongURL)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "could not parse request URL")
		return
	}

	shortURLHash, err := hashCollisionDetection(apiCfg.DB, url.String(), 1, r.Context())

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "could not resolve hash collision")
		return
	}

	now := time.Now()
	shortenedURL, err := apiCfg.DB.CreateURL(r.Context(), database.CreateURLParams{
		LongUrl:   url.String(),
		ShortUrl:  shortURLHash,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
	})

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "could not create short URL in database")
		return
	}

	respondWithJSON(w, http.StatusCreated, LongURLResponse{
		ShortURL: shortenedURL.ShortUrl,
	})
}

func (apiCfg *apiConfig) getShortURL(w http.ResponseWriter, r *http.Request) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	cacheVal, err := apiCfg.RDB.Get(r.Context(), query).Result()

	switch {
	case err == redis.Nil:
		log.Printf("cache miss, key %s does not exists, writing to redis", query)

		row, err := apiCfg.DB.SelectURL(r.Context(), query)

		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "database error")
			return
		}

		err = apiCfg.RDB.Set(r.Context(), query, row.LongUrl, (time.Hour * 1)).Err()

		if err != nil {
			log.Printf("could not write to redis cache %s", err)
		}

		http.Redirect(w, r, row.LongUrl, http.StatusMovedPermanently)
		return

	case err != nil:
		log.Println(err)

		row, err := apiCfg.DB.SelectURL(r.Context(), query)

		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "database error")
			return
		}

		http.Redirect(w, r, row.LongUrl, http.StatusMovedPermanently)
		return

	case cacheVal == "":
		log.Printf("key %s does not have a value", query)

		row, err := apiCfg.DB.SelectURL(r.Context(), query)

		if err != nil {
			log.Println(err)
			respondWithError(w, http.StatusInternalServerError, "database error")
			return
		}

		http.Redirect(w, r, row.LongUrl, http.StatusMovedPermanently)
		return
	}

	log.Printf("cache hit for key %s", cacheVal)
	http.Redirect(w, r, cacheVal, http.StatusMovedPermanently)
}

func (apiCfg *apiConfig) deleteShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadRequest, "no short url supplied")
		return
	}

	err := apiCfg.DB.DeleteURL(r.Context(), database.DeleteURLParams{
		UserID:   user.ID,
		ShortUrl: query,
	})

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "could not delete short url")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (apiCfg *apiConfig) putShortURL(w http.ResponseWriter, r *http.Request, user database.User) {
	query := r.PathValue("shortUrl")

	if query == "" {
		respondWithError(w, http.StatusBadGateway, "no short url supplied")
		return
	}

	payload := LongURLRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
	}

	err = apiCfg.DB.UpdateShortURL(r.Context(), database.UpdateShortURLParams{
		UserID:   user.ID,
		ShortUrl: query,
		LongUrl:  payload.LongURL,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update long url")
		return
	}

	err = apiCfg.RDB.Set(r.Context(), query, payload.LongURL, (time.Hour * 1)).Err()

	if err != nil {
		log.Println(err)
	}

	respondWithJSON(w, http.StatusOK, ShortURLUpdateResponse{
		LongURL:  payload.LongURL,
		ShortURL: query,
	})
}

func (apiCfg *apiConfig) postAPIUsers(w http.ResponseWriter, r *http.Request) {
	payload := APIUserRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "could not parse request")
		return
	}

	domainUser, err := user.NewUser(payload.Email, payload.Password)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := apiCfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:     domainUser.Email().Address,
		Password:  domainUser.PasswordHash(),
		CreatedAt: domainUser.CreatedAt(),
		UpdatedAt: domainUser.UpdatedAt(),
	})

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, "could not create user in database")
	}

	respondWithJSON(w, http.StatusCreated, APIUserResponseNoToken{
		ID:    user.ID,
		Email: user.Email,
	})
}

func (apiCfg *apiConfig) postAPILogin(w http.ResponseWriter, r *http.Request) {
	payload := APIUserRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not parse request")
		return
	}

	if payload.Email == "" || payload.Password == "" {
		respondWithError(w, http.StatusBadRequest, "invalid parameters for user login")
	}

	user, err := apiCfg.DB.SelectUser(r.Context(), payload.Email)

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

	signedToken, err := token.SignedString([]byte(apiCfg.JWTSecret))

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

	err = apiCfg.DB.UserTokenRefresh(r.Context(), database.UserTokenRefreshParams{
		RefreshToken:           sql.NullString{String: refreshToken, Valid: true},
		RefreshTokenRevokeDate: sql.NullTime{Time: time.Now().Add(60 * (24 * time.Hour)), Valid: true},
		ID:                     user.ID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can not update user with refresh token")
		return
	}

	respondWithJSON(w, http.StatusFound, APIUsersResponse{
		ID:           user.ID,
		Email:        user.Email,
		Token:        signedToken,
		RefreshToken: refreshToken,
	})
}

func (apiCfg *apiConfig) putAPIUsers(w http.ResponseWriter, r *http.Request, user database.User) {
	payload := APIUserRequest{}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "incorrect parameters for user update request")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "bad password supplied from client")
		return
	}

	err = apiCfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:     payload.Email,
		Password:  string(passwordHash),
		ID:        user.ID,
		UpdatedAt: time.Now(),
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update user in database")
	}

	respondWithJSON(w, http.StatusOK, APIUserResponseNoToken{
		Email: payload.Email,
		ID:    user.ID,
	})
}

func (apiCfg *apiConfig) postAPIRefresh(w http.ResponseWriter, r *http.Request) {
	requestToken, err := extractAuthTokenFromRequest(r)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := apiCfg.DB.SelectUserByRefreshToken(r.Context(), sql.NullString{String: requestToken, Valid: true})

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "can not refresh token no user found")
		return
	}

	if time.Now().After(user.RefreshTokenRevokeDate.Time) {
		respondWithError(w, http.StatusUnauthorized, "refresh token expired, please login again")
		return
	}

	// TODO: We do this twice in the codebase, if we do it a third time pull this out to a general JWT issue function
	registeredClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "url-short-auth",
		Subject:   strconv.Itoa(int(user.ID)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, registeredClaims)

	signedToken, err := token.SignedString([]byte(apiCfg.JWTSecret))

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can not create JWT")
		return
	}

	respondWithJSON(w, http.StatusCreated, APIUsersRefreshResponse{
		Token: signedToken,
	})
}
