package users

import (
	"encoding/json"
	"log"
	"net/http"

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
