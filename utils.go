package main

import (
	"crypto/rand"
	"encoding/base64"
)

// RandomString generates random bytes and then encodes them using base64
// which guarantees they are URL-safe. The resultant output is not necessarily
// a valid base-64 string.
func RandomString(length int) string {
	bytes := make([]byte, length*2)
	rand.Read(bytes)
	se := base64.StdEncoding.EncodeToString(bytes)
	return se[0:length]
}
