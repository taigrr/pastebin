package main

import (
	"time"
)

// Config holds the server configuration.
type Config struct {
	Bind   string
	FQDN   string
	Expiry time.Duration
}
