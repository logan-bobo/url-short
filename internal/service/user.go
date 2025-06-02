package service

import (
	"context"

	"url-short/internal/domain/user"
	"url-short/internal/repository"
)

type UserService interface {
	CreateUser(ctx context.Context, request user.CreateUserRequest) (*user.User, error)
	//UpdateUser()
	//LoginUser()
	//GetUserRefreshToken()
}

type UserServiceImpl struct {
	userRepo repository.UserRepository
}

func NewUserServiceImpl(r repository.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{
		userRepo: r,
	}
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, request user.CreateUserRequest) (*user.User, error) {
	res, err := s.userRepo.CreateUser(ctx, request)
	if err != nil {
		return nil, err
	}

	return res, nil

}
