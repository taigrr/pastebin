package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		err := r.ParseForm()
		require.NoError(t, err)
		blob := r.FormValue("blob")
		assert.Equal(t, "test content", blob)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cli := NewClient(server.URL, false)
	err := cli.Paste(strings.NewReader("test content"))
	assert.NoError(t, err)
}

func TestPasteServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cli := NewClient(server.URL, false)
	err := cli.Paste(strings.NewReader("test content"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response")
}

func TestPasteInvalidURL(t *testing.T) {
	cli := NewClient("http://invalid.invalid.invalid:99999", false)
	err := cli.Paste(strings.NewReader("test content"))
	assert.Error(t, err)
}

func TestNewClient(t *testing.T) {
	cli := NewClient("http://example.com", true)
	assert.Equal(t, "http://example.com", cli.url)
	assert.True(t, cli.insecure)
}
