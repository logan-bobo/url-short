package user

import (
	"errors"
	"net/mail"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	id                     int32
	email                  mail.Address
	passwordHash           []byte
	createdAt              time.Time
	updatedAt              time.Time
	refreshToken           string
	refreshTokenRevokeDate time.Time
}

func NewUser(email, password string) (*User, error) {
	if email == "" {
		return nil, errors.New("could not create user: empty email")
	}

	parsedEmail, err := mail.ParseAddress(email)
	if err != nil {
		return nil, errors.New("could not create user: invalid email")
	}

	if password == "" {
		return nil, errors.New("could not create user: empty password")
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return nil, errors.New("could not create user: failure to hash password")
	}

	now := time.Now()

	return &User{
		email:        *parsedEmail,
		passwordHash: passwordHash,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func (u *User) ID() int32 {
	return u.id
}

func (u *User) SetID(id int32) {
	u.id = id
}

func (u *User) Email() string {
	return u.email.Address
}

func (u *User) PasswordHash() string {
	return string(u.passwordHash)
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

func (u *User) RefreshToken() string {
	return u.refreshToken
}

func (u *User) RefreshTokenRevokeDate() time.Time {
	return u.refreshTokenRevokeDate
}
