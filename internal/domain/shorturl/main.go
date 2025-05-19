package shorturl

import (
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"strings"
)

type ShortUrl struct {
	ShortUrl *url.URL
	LongUrl  *url.URL
}

func NewShortUrl(longUrl string) (*ShortUrl, error) {
	parsed, err := url.ParseRequestURI(longUrl)
	if err != nil {
		return nil, err
	}

	return &ShortUrl{
		LongUrl: parsed,
	}, nil
}

const urlHashPostfix = "Xa1"

func Hash(inputString string, postfixCount int) string {
	hash := md5.Sum([]byte(inputString + strings.Repeat(urlHashPostfix, postfixCount)))

	return hex.EncodeToString(hash[:])
}

func Shorten(hashToShorten string) string {
	if len(hashToShorten) < 8 {
		return hashToShorten
	}

	return hashToShorten[:7]
}
