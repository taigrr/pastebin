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

func TestPasteOversized(t *testing.T) {
	server := newTestServer()

	// Create a paste larger than maxPasteSize (1 MB)
	largeBlob := strings.Repeat("x", maxPasteSize+1)
	formData := url.Values{}
	formData.Set("blob", largeBlob)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestViewHTMLRender(t *testing.T) {
	server := newTestServer()

	server.store.Set("htmlview", "rendered content", 0)

	req := httptest.NewRequest(http.MethodGet, "/p/htmlview", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rec.Body.String(), "rendered content")
	assert.Contains(t, rec.Body.String(), "htmlview")
}

func TestStatsEmptyStore(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/debug/stats", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"item_count":0`)
}

func TestStatsMultipleItems(t *testing.T) {
	server := newTestServer()

	server.store.Set("a", "1", 0)
	server.store.Set("b", "2", 0)
	server.store.Set("c", "3", 0)

	req := httptest.NewRequest(http.MethodGet, "/debug/stats", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"item_count":3`)
}

func TestPasteNoFormField(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteResponseBody(t *testing.T) {
	server := newTestServer()

	server.store.Set("delresp", "content", 0)

	req := httptest.NewRequest(http.MethodDelete, "/p/delresp", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Deleted", rec.Body.String())
}

func TestDownloadContentHeaders(t *testing.T) {
	server := newTestServer()

	server.store.Set("dlheader", "file content here", 0)

	req := httptest.NewRequest(http.MethodGet, "/download/dlheader", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "attachment; filename=dlheader", rec.Header().Get("Content-Disposition"))
}

func TestNegotiateContentTypeDefault(t *testing.T) {
	server := newTestServer()

	// No Accept header defaults to plain text
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "pastebin service")
}

func TestPasteRoundTripSpecialChars(t *testing.T) {
	server := newTestServer()

	specialContent := "line1\nline2\n<script>alert('xss')</script>\n日本語テスト"

	formData := url.Values{}
	formData.Set("blob", specialContent)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/plain")
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	parts := strings.Split(rec.Body.String(), "/p/")
	require.Len(t, parts, 2)
	pasteID := parts[1]

	viewReq := httptest.NewRequest(http.MethodGet, "/p/"+pasteID, nil)
	viewReq.Header.Set("Accept", "text/plain")
	viewRec := httptest.NewRecorder()

	server.mux.ServeHTTP(viewRec, viewReq)

	assert.Equal(t, http.StatusOK, viewRec.Code)
	assert.Equal(t, specialContent, viewRec.Body.String())
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
