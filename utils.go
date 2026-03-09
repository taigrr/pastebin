package main

import (
	"crypto/rand"
	"encoding/base64"
)

// RandomString generates a URL-safe random string of the specified length.
// It reads random bytes and encodes them using base64 URL encoding.
// The resulting string is truncated to the requested length.
func RandomString(length int) string {
	rawBytes := make([]byte, length*2)
	_, _ = rand.Read(rawBytes)
	encoded := base64.URLEncoding.EncodeToString(rawBytes)
	return encoded[:length]
}
