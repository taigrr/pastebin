package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// RandomString generates a URL-safe random string of the specified length.
// It reads random bytes and encodes them using base64 URL encoding.
// The resulting string is truncated to the requested length.
// It panics if the system's cryptographic random number generator fails,
// which indicates a fundamental system issue.
func RandomString(length int) string {
	if length <= 0 {
		return ""
	}
	rawBytes := make([]byte, length*2)
	if _, err := rand.Read(rawBytes); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	encoded := base64.URLEncoding.EncodeToString(rawBytes)
	return encoded[:length]
}
