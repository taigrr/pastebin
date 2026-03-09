package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestLoadConfigDefaults(t *testing.T) {
	InitConfig()
	cfg := LoadConfig()

	assert.Equal(t, defaultBind, cfg.Bind)
	assert.Equal(t, defaultFQDN, cfg.FQDN)
	assert.Equal(t, defaultExpiry, cfg.Expiry)
}

func TestReadConfigFileTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `bind = "127.0.0.1:9090"
fqdn = "paste.example.com"
expiry = "10m"
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	InitConfig()
	require.NoError(t, ReadConfigFile(configPath))

	cfg := LoadConfig()
	assert.Equal(t, "127.0.0.1:9090", cfg.Bind)
	assert.Equal(t, "paste.example.com", cfg.FQDN)
	assert.Equal(t, 10*time.Minute, cfg.Expiry)
}

func TestReadConfigFileJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	content := `{"bind": "0.0.0.0:3000", "fqdn": "example.org", "expiry": "30m"}`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	InitConfig()
	require.NoError(t, ReadConfigFile(configPath))

	cfg := LoadConfig()
	assert.Equal(t, "0.0.0.0:3000", cfg.Bind)
	assert.Equal(t, "example.org", cfg.FQDN)
	assert.Equal(t, 30*time.Minute, cfg.Expiry)
}

func TestReadConfigFileYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `bind: "10.0.0.1:4000"
fqdn: "yaml.example.com"
expiry: "1h"
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	InitConfig()
	require.NoError(t, ReadConfigFile(configPath))

	cfg := LoadConfig()
	assert.Equal(t, "10.0.0.1:4000", cfg.Bind)
	assert.Equal(t, "yaml.example.com", cfg.FQDN)
	assert.Equal(t, time.Hour, cfg.Expiry)
}

func TestReadConfigFileEmpty(t *testing.T) {
	InitConfig()
	// Empty path should be a no-op.
	assert.NoError(t, ReadConfigFile(""))
}

func TestReadConfigFileNotFound(t *testing.T) {
	InitConfig()
	err := ReadConfigFile("/nonexistent/config.toml")
	assert.Error(t, err)
}

func TestReadConfigFileUnsupportedExtension(t *testing.T) {
	InitConfig()
	err := ReadConfigFile("/tmp/config.xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file extension")
}

func TestDetectConfigType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
		wantErr  bool
	}{
		{"config.toml", "toml", false},
		{"config.yaml", "yaml", false},
		{"config.yml", "yaml", false},
		{"config.json", "json", false},
		{"config.xml", "", true},
		{"/path/to/my.toml", "toml", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got, err := detectConfigType(tc.path)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}
