package user

import (
	"errors"
	"net/mail"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id                     int32
	Email                  string
	PasswordHash           []byte
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Token                  string
	RefreshToken           string
	RefreshTokenRevokeDate time.Time
}

var (
	ErrEmptyEmail          = errors.New("empty email")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrEmptyPassword       = errors.New("empty password")
	ErrInvalidPassword     = errors.New("invalid password")
	ErrInvalidLoginRequest = errors.New("email and password must not be empty")
	ErrUserNotFound        = errors.New("user could not be found")
	ErrUnexpectedError     = errors.New("unexpected server error")
	ErrDuplicateUSer       = errors.New("user already exists")
)

func NewUser(email, password string) (*User, error) {
	if email == "" {
		return nil, ErrEmptyEmail
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return nil, ErrInvalidEmail
	}

	if password == "" {
		return nil, ErrEmptyPassword
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return nil, ErrInvalidPassword
	}

	return &User{
		Email:        email,
		PasswordHash: passwordHash,
	}, nil
}

func (u *User) GetPasswordHash() string {
	return string(u.PasswordHash)
}

type CreateUserRequest struct {
	Email        string
	PasswordHash string
}

func NewCreateUserRequest(email, password string) (*CreateUserRequest, error) {
	user, err := NewUser(email, password)
	if err != nil {
		return nil, err
	}

	return &CreateUserRequest{
		Email:        user.Email,
		PasswordHash: user.GetPasswordHash(),
	}, nil
}

type LoginUserRequest struct {
	Email    string
	Password string
}

func NewLoginUserRequest(email, password string) (*LoginUserRequest, error) {
	if email == "" || password == "" {
		return nil, ErrInvalidLoginRequest
	}

	return &LoginUserRequest{
		Email:    email,
		Password: password,
	}, nil
}

type UpdateUserRequest struct {
	Id              int32
	Email           string
	NewPasswordHash string
}

func NewUpdateUserRequest(email, password string, userId int32) (*UpdateUserRequest, error) {
	user, err := NewUser(email, password)
	if err != nil {
		return nil, err
	}

	return &UpdateUserRequest{
		Id:              userId,
		Email:           user.Email,
		NewPasswordHash: user.GetPasswordHash(),
	}, nil
}
