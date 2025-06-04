package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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

	loginUserRequest, err := user.NewLoginUserRequest(payload.Email, payload.Password)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := handler.userService.LoginUser(r.Context(), *loginUserRequest)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could be lots of errors need to handle")
		return
	}

	respondWithJSON(w, http.StatusFound, loginUserHTTPResponseBody{
		ID:           res.Id,
		Email:        res.Email,
		Token:        res.Token,
		RefreshToken: res.RefreshToken,
	})
}

type refreshAccessTokenHTTPResponseBody struct {
	AccessToken string `json:"token"`
}

func (handler *userHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	requestToken, err := ExtractAuthTokenFromRequest(r)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := handler.userService.RefreshAccessToken(r.Context(), requestToken)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not refresh access token")
		return
	}

	respondWithJSON(w, http.StatusCreated, refreshAccessTokenHTTPResponseBody{
		AccessToken: user.Token,
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
