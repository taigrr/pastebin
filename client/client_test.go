package client

import (
	"bytes"
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
		_, _ = w.Write([]byte(r.Host + "/p/abc123"))
	}))
	defer server.Close()

	var output bytes.Buffer
	cli := NewClient(server.URL, false).WithOutput(&output)
	err := cli.Paste(strings.NewReader("test content"))
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "/p/abc123")
}

func TestPasteRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/p/redirect123")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	var output bytes.Buffer
	cli := NewClient(server.URL, false).WithOutput(&output)
	err := cli.Paste(strings.NewReader("redirect content"))
	assert.NoError(t, err)
	assert.Equal(t, "/p/redirect123", output.String())
}

func TestPasteServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var output bytes.Buffer
	cli := NewClient(server.URL, false).WithOutput(&output)
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
	assert.Equal(t, "http://example.com", cli.serviceURL)
	assert.True(t, cli.insecure)
}

func TestWithOutput(t *testing.T) {
	var buf bytes.Buffer
	cli := NewClient("http://example.com", false).WithOutput(&buf)
	assert.Equal(t, &buf, cli.output)
}
