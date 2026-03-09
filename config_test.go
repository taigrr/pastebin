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

func TestConfigDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		expiry time.Duration
		bind   string
		fqdn   string
	}{
		{
			name:   "short expiry",
			config: Config{Expiry: 1 * time.Minute, Bind: ":8080", FQDN: "paste.example.com"},
			expiry: 1 * time.Minute,
			bind:   ":8080",
			fqdn:   "paste.example.com",
		},
		{
			name:   "long expiry",
			config: Config{Expiry: 24 * time.Hour, Bind: "127.0.0.1:3000", FQDN: "localhost"},
			expiry: 24 * time.Hour,
			bind:   "127.0.0.1:3000",
			fqdn:   "localhost",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expiry, tc.config.Expiry)
			assert.Equal(t, tc.bind, tc.config.Bind)
			assert.Equal(t, tc.fqdn, tc.config.FQDN)
		})
	}
}
