package shorturl

import (
	"net/url"
	"time"
)

type URL struct {
	ID int32
	// TODO: Make ShortURL type
	ShortURL  string
	LongURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int32
}

func NewLongURL(URL string) (*string, error) {
	_, err := url.ParseRequestURI(URL)
	if err != nil {
		return nil, err
	}

	return &URL, nil
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

type URLError struct {
	Code    string
	Message string
}

type URLValidationError struct {
	URLError
}

func NewValidationError(code, msg string) *URLValidationError {
	return &URLValidationError{URLError{
		Code:    code,
		Message: msg,
	}}
}

type URLNotFoundError struct {
	URLError
}

func NewNotFoundError(code, msg string) *URLNotFoundError {
	return &URLNotFoundError{URLError{
		Code:    code,
		Message: msg,
	}}
}
