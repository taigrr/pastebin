package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer() *Server {
	cfg := Config{
		Bind:   "127.0.0.1:0",
		FQDN:   "localhost",
		Expiry: 5 * time.Minute,
	}
	return NewServer(cfg)
}

func TestIndexHandlerHTML(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Paste")
}

func TestIndexHandlerPlain(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/plain")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "pastebin service")
}

func TestPasteAndView(t *testing.T) {
	server := newTestServer()

	// Create a paste
	formData := url.Values{}
	formData.Set("blob", "hello world")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/plain")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	pasteURL := rec.Body.String()
	assert.Contains(t, pasteURL, "/p/")

	// Extract paste ID from URL
	parts := strings.Split(pasteURL, "/p/")
	require.Len(t, parts, 2)
	pasteID := parts[1]

	// View the paste as plain text
	viewReq := httptest.NewRequest(http.MethodGet, "/p/"+pasteID, nil)
	viewReq.Header.Set("Accept", "text/plain")
	viewRec := httptest.NewRecorder()

	server.mux.ServeHTTP(viewRec, viewReq)

	assert.Equal(t, http.StatusOK, viewRec.Code)
	assert.Equal(t, "hello world", viewRec.Body.String())
}

func TestPasteAndViewHTML(t *testing.T) {
	server := newTestServer()

	// Create a paste via HTML (should redirect)
	formData := url.Values{}
	formData.Set("blob", "test content")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "/p/")
}

func TestPasteEmptyBlob(t *testing.T) {
	server := newTestServer()

	formData := url.Values{}
	formData.Set("blob", "")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestViewNotFound(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/p/nonexistent", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeletePaste(t *testing.T) {
	server := newTestServer()

	// Create a paste
	server.store.Set("testid", "test content", 0)

	// Delete it
	req := httptest.NewRequest(http.MethodDelete, "/p/testid", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify it's gone
	_, found := server.store.Get("testid")
	assert.False(t, found)
}

func TestDeletePasteViaPost(t *testing.T) {
	server := newTestServer()

	server.store.Set("testid2", "test content", 0)

	req := httptest.NewRequest(http.MethodPost, "/delete/testid2", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	_, found := server.store.Get("testid2")
	assert.False(t, found)
}

func TestDeleteNotFound(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/p/nonexistent", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDownloadPaste(t *testing.T) {
	server := newTestServer()

	server.store.Set("dltest", "download content", 0)

	req := httptest.NewRequest(http.MethodGet, "/download/dltest", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "attachment; filename=dltest", rec.Header().Get("Content-Disposition"))
	assert.Contains(t, rec.Body.String(), "download content")
}

func TestDownloadNotFound(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/download/nonexistent", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestStatsHandler(t *testing.T) {
	server := newTestServer()

	server.store.Set("item1", "content", 0)

	req := httptest.NewRequest(http.MethodGet, "/debug/stats", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rec.Body.String(), "item_count")
}

func TestViewWithTabs(t *testing.T) {
	server := newTestServer()

	server.store.Set("tabtest", "line1\tindented", 0)

	req := httptest.NewRequest(http.MethodGet, "/p/tabtest", nil)
	req.Header.Set("Accept", "text/plain")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "line1    indented", rec.Body.String())
}
