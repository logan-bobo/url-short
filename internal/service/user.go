package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"url-short/internal/domain/user"
	"url-short/internal/repository"
)

type UserService interface {
	CreateUser(ctx context.Context, request user.CreateUserRequest) (*user.User, error)
	LoginUser(ctx context.Context, request user.LoginUserRequest) (*user.User, error)
	RefreshAccessToken(ctx context.Context, token string) (*user.User, error)
	UpdateUser(ctx context.Context, request user.UpdateUserRequest) (*user.User, error)
	ValidateUserJWT(ctx context.Context, requestToken string) (*user.User, error)
}

type UserServiceImpl struct {
	userRepo  repository.UserRepository
	JWTSecret string
}

func NewUserServiceImpl(r repository.UserRepository, jwtSecret string) *UserServiceImpl {
	return &UserServiceImpl{
		userRepo:  r,
		JWTSecret: jwtSecret,
	}
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, request user.CreateUserRequest) (*user.User, error) {
	res, err := s.userRepo.CreateUser(ctx, request)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *UserServiceImpl) LoginUser(ctx context.Context, request user.LoginUserRequest) (*user.User, error) {
	res, err := s.userRepo.SelectUser(ctx, request.Email)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(res.PasswordHash, []byte(request.Password))
	if err != nil {
		return nil, user.ErrInvalidPassword
	}

	registeredClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "url-short-auth",
		Subject:   strconv.Itoa(int(res.Id)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, registeredClaims)

	signedToken, err := token.SignedString([]byte(s.JWTSecret))
	if err != nil {
		return nil, user.ErrUnexpectedError
	}

	byteSlice := make([]byte, 32)
	_, err = rand.Read(byteSlice)
	refreshToken := hex.EncodeToString(byteSlice)
	if err != nil {
		return nil, user.ErrUnexpectedError
	}

	err = s.userRepo.UpdateRefreshToken(ctx, refreshToken, res.Id)
	if err != nil {
		return nil, err
	}

	res.RefreshToken = refreshToken
	res.Token = signedToken

	return res, nil
}

func (s *UserServiceImpl) RefreshAccessToken(ctx context.Context, refreshToken string) (*user.User, error) {
	user, err := s.userRepo.SelectUserByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	if time.Now().After(user.RefreshTokenRevokeDate) {
		return nil, errors.New("refresh token expired, please login again")
	}

	registeredClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "url-short-auth",
		Subject:   strconv.Itoa(int(user.Id)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, registeredClaims)

	signedToken, err := token.SignedString([]byte(s.JWTSecret))

	if err != nil {
		return nil, err
	}

	user.Token = signedToken

	return user, nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, request user.UpdateUserRequest) (*user.User, error) {
	user, err := s.userRepo.UpdateUser(ctx, request)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserServiceImpl) ValidateUserJWT(ctx context.Context, requestToken string) (*user.User, error) {
	claims := jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(
		requestToken,
		&claims,
		func(token *jwt.Token) (any, error) { return []byte(s.JWTSecret), nil },
	)
	if err != nil {
		return nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return nil, err
	}

	if issuer != "url-short-auth" {
		return nil, jwt.ErrTokenInvalidIssuer
	}

	userID, err := token.Claims.GetSubject()
	if err != nil {
		return nil, err
	}

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		return nil, err
	}

	validatedUser, err := s.userRepo.SelectUserByID(ctx, int32(userIDInt))
	if err != nil {
		return nil, err
	}

	return validatedUser, nil
}
