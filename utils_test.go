package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomStringLength(t *testing.T) {
	for _, length := range []int{1, 4, 8, 16, 32} {
		result := RandomString(length)
		assert.Len(t, result, length, "RandomString(%d) should return string of length %d", length, length)
	}
}

func TestRandomStringUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		result := RandomString(pasteIDLength)
		require.False(t, seen[result], "RandomString produced duplicate: %s", result)
		seen[result] = true
	}
}

func TestRandomStringURLSafe(t *testing.T) {
	for range 50 {
		result := RandomString(32)
		// base64 URL encoding uses A-Z, a-z, 0-9, -, _
		for _, char := range result {
			isValid := (char >= 'A' && char <= 'Z') ||
				(char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '_' || char == '='
			assert.True(t, isValid, "RandomString contains invalid URL character: %c", char)
		}
	}
}
