package shorturl

import (
	"fmt"
	"net/url"
	"time"
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
		return nil, NewURLValidationError(fmt.Sprintf("could not validate url, %v", err))
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

type URLError struct {
	Message string
}

func (e *URLError) Error() string {
	return fmt.Sprintf("%s", e.Message)
}

type URLValidationError struct {
	URLError
}

func NewURLValidationError(msg string) *URLValidationError {
	return &URLValidationError{URLError{
		Message: msg,
	}}
}

type URLNotFoundError struct {
	URLError
}

func NewURLNotFoundError() *URLNotFoundError {
	return &URLNotFoundError{URLError{
		Message: "url could not be found",
	}}
}

type URLCreationError struct {
	URLError
}

func NewURLCreationError(msg string) *URLCreationError {
	return &URLCreationError{URLError{
		Message: msg,
	}}
}
