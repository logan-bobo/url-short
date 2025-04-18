package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

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

func (apiCfg *apiConfig) putAPIUsers(w http.ResponseWriter, r *http.Request, authUser database.User) {
	payload := APIUserRequest{}

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
	domainUser.SetID(authUser.ID)

	err = apiCfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:     domainUser.Email(),
		Password:  domainUser.PasswordHash(),
		ID:        domainUser.ID(),
		UpdatedAt: time.Now(),
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not update user in database")
	}

	respondWithJSON(w, http.StatusOK, APIUserResponseNoToken{
		Email: payload.Email,
		ID:    domainUser.ID(),
	})
}

func (apiCfg *apiConfig) postAPIRefresh(w http.ResponseWriter, r *http.Request) {
	requestToken, err := extractAuthTokenFromRequest(r)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// This should be moved down to a repo layer with a service layer in the middle
	// the current handler should go API -> Service -> Repo but right now it does not make sense
	// to move this hanlder to use the user domain. This should however be refactored.
	user, err := apiCfg.DB.SelectUserByRefreshToken(r.Context(), sql.NullString{String: requestToken, Valid: true})

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

	signedToken, err := token.SignedString([]byte(apiCfg.JWTSecret))

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "can not create JWT")
		return
	}

	respondWithJSON(w, http.StatusCreated, APIUsersRefreshResponse{
		Token: signedToken,
	})
}
