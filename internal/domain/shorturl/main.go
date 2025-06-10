package shorturl

import (
	"errors"
	"net/url"
	"time"
)

var (
	ErrURLNotFound     = errors.New("url could not be found")
	ErrURLValidation   = errors.New("could not validate url")
	ErrUnexpectedError = errors.New("unexpected server error")
)

type URL struct {
	ID        int32
	ShortURL  string
	LongURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int32
}

func NewLongURL(URL string) (*string, error) {
	_, err := url.ParseRequestURI(URL)
	if err != nil {
		return nil, ErrURLValidation
	}

	return &URL, nil
}

func NewShortURL(URL string) (string, error) {
	if URL == "" {
		return "", ErrURLValidation
	}

	return URL, nil
}

func NewUrl(longURL string) (*URL, error) {
	parsed, err := NewLongURL(longURL)
	if err != nil {
		return nil, err
	}

	return &URL{
		LongURL: *parsed,
	}, nil
}

type CreateURLRequest struct {
	UserID   int32
	LongURL  string
	ShortURL string
}

func NewCreateURLRequest(userID int32, URL string) (*CreateURLRequest, error) {
	parsed, err := NewLongURL(URL)
	if err != nil {
		return nil, err
	}

	return &CreateURLRequest{
		UserID:  userID,
		LongURL: *parsed,
	}, nil
}

type DeleteURLRequest struct {
	UserID   int32
	ShortURL string
}

func NewDeleteURLRequest(userID int32, URL string) *DeleteURLRequest {
	return &DeleteURLRequest{
		UserID:   userID,
		ShortURL: URL,
	}
}

type UpdateURLRequest struct {
	UserID   int32
	ShortURL string
	LongURL  string
}

func NewUpdateURLRequest(userID int32, shortURL string, longURL string) *UpdateURLRequest {
	return &UpdateURLRequest{
		UserID:   userID,
		ShortURL: shortURL,
		LongURL:  longURL,
	}
}
