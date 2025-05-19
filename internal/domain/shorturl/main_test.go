package shorturl

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
)

func TestHash(t *testing.T) {
	url := "www.test.com/shop"

	t.Run("test URL hash with no collision", func(t *testing.T) {
		got := Hash(url, 0)
		hash := md5.Sum([]byte(url))
		want := hex.EncodeToString(hash[:])

		if got != want {
			t.Errorf("got %q, wanted %q given %q", got, want, url)
		}
	})

	t.Run("test URL hash with collision", func(t *testing.T) {
		got := Hash(url, 1)
		hash := md5.Sum([]byte(url + urlHashPostfix))
		want := hex.EncodeToString(hash[:])

		if got != want {
			t.Errorf("got %q, want %q, given %q", got, want, url)
		}
	})
}

func TestShort(t *testing.T) {
	hashLong := "asdfghjkl"
	shortHash := "xa1"

	t.Run("test hash shoren with large hash", func(t *testing.T) {
		got := Shorten(hashLong)
		want := hashLong[:7]

		if got != want {
			t.Errorf("got %q, wanted %q, given %q", got, want, hashLong)
		}
	})

	t.Run("test hash with short hash", func(t *testing.T) {
		got := Shorten(shortHash)
		want := shortHash

		if got != want {
			t.Errorf("got %q, want %q, given %q", got, want, shortHash)
		}
	})
}
