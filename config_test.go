package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestZeroConfig(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, time.Duration(0), cfg.Expiry)
	assert.Equal(t, "", cfg.FQDN)
	assert.Equal(t, "", cfg.Bind)
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Expiry: 30 * time.Minute,
		FQDN:   "https://localhost",
		Bind:   "0.0.0.0:8000",
	}
	assert.Equal(t, 30*time.Minute, cfg.Expiry)
	assert.Equal(t, "https://localhost", cfg.FQDN)
	assert.Equal(t, "0.0.0.0:8000", cfg.Bind)
}
