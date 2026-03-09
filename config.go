package main

import (
	"fmt"
	"time"

	"github.com/taigrr/jety"
)

const (
	configKeyBind   = "bind"
	configKeyFQDN   = "fqdn"
	configKeyExpiry = "expiry"

	defaultBind   = "0.0.0.0:8000"
	defaultFQDN   = "localhost"
	defaultExpiry = 5 * time.Minute

	envPrefix = "PASTEBIN_"
)

// Config holds the server configuration.
type Config struct {
	Bind   string
	FQDN   string
	Expiry time.Duration
}

// InitConfig sets up jety defaults and the environment variable prefix.
// Call this before reading a config file or loading configuration.
func InitConfig() {
	jety.SetDefault(configKeyBind, defaultBind)
	jety.SetDefault(configKeyFQDN, defaultFQDN)
	jety.SetDefault(configKeyExpiry, defaultExpiry)
	jety.SetEnvPrefix(envPrefix)
}

// ReadConfigFile reads configuration from the specified file path.
// Supported formats are JSON, TOML, and YAML (detected by extension).
func ReadConfigFile(path string) error {
	if path == "" {
		return nil
	}
	jety.SetConfigFile(path)

	configType, err := detectConfigType(path)
	if err != nil {
		return err
	}
	if err := jety.SetConfigType(configType); err != nil {
		return fmt.Errorf("setting config type: %w", err)
	}
	if err := jety.ReadInConfig(); err != nil {
		return fmt.Errorf("reading config file %s: %w", path, err)
	}
	return nil
}

// detectConfigType returns the config format string based on file extension.
func detectConfigType(path string) (string, error) {
	for _, ext := range []struct {
		suffix     string
		configType string
	}{
		{".toml", "toml"},
		{".yaml", "yaml"},
		{".yml", "yaml"},
		{".json", "json"},
	} {
		if len(path) > len(ext.suffix) && path[len(path)-len(ext.suffix):] == ext.suffix {
			return ext.configType, nil
		}
	}
	return "", fmt.Errorf("unsupported config file extension: %s", path)
}

// LoadConfig builds a Config from the current jety state.
// Precedence: Set() > environment variables > config file > defaults.
func LoadConfig() Config {
	return Config{
		Bind:   jety.GetString(configKeyBind),
		FQDN:   jety.GetString(configKeyFQDN),
		Expiry: jety.GetDuration(configKeyExpiry),
	}
}
