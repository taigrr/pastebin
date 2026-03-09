package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/patrickmn/go-cache"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/css/*.css
var staticFS embed.FS

const (
	contentTypeHTML  = "text/html"
	contentTypePlain = "text/plain"
	contentTypeJSON  = "application/json; charset=utf-8"

	headerAccept             = "Accept"
	headerContentDisposition = "Content-Disposition"
	headerContentType        = "Content-Type"
	headerContentLength      = "Content-Length"

	formFieldBlob = "blob"

	pasteIDLength = 8

	// maxPasteSize limits the maximum size of a paste body (1 MB).
	maxPasteSize = 1 << 20

	// maxCollisionRetries is the number of times to retry generating a paste ID on collision.
	maxCollisionRetries = 3

	// shutdownTimeout is the maximum time to wait for in-flight requests during shutdown.
	shutdownTimeout = 10 * time.Second

	statusHealthy = "healthy"
)

// Server holds the pastebin HTTP server state.
type Server struct {
	config    Config
	store     *cache.Cache
	templates *template.Template
	mux       *http.ServeMux
}

// NewServer creates and configures a new pastebin Server.
func NewServer(config Config) *Server {
	server := &Server{
		config: config,
		mux:    http.NewServeMux(),
		store:  cache.New(config.Expiry, config.Expiry*2),
	}

	server.templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

	server.initRoutes()

	return server
}

// ListenAndServe starts the HTTP server on the configured bind address.
// It handles graceful shutdown on SIGINT and SIGTERM.
func (s *Server) ListenAndServe() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:    s.config.Bind,
		Handler: s.mux,
	}

	errChan := make(chan error, 1)
	go func() {
		log.Printf("pastebin listening on %s", s.config.Bind)
		errChan <- server.ListenAndServe()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		log.Println("shutting down gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func (s *Server) initRoutes() {
	cssFS, err := fs.Sub(staticFS, "static/css")
	if err != nil {
		log.Fatalf("failed to create sub filesystem for css: %v", err)
	}
	s.mux.Handle("GET /css/", http.StripPrefix("/css/", http.FileServer(http.FS(cssFS))))

	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("POST /{$}", s.handlePaste)
	s.mux.HandleFunc("GET /p/{uuid}", s.handleView)
	s.mux.HandleFunc("DELETE /p/{uuid}", s.handleDelete)
	s.mux.HandleFunc("POST /delete/{uuid}", s.handleDelete)
	s.mux.HandleFunc("GET /download/{uuid}", s.handleDownload)
	s.mux.HandleFunc("GET /debug/stats", s.handleStats)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
}

// templateData holds data passed to HTML templates.
type templateData struct {
	Blob string
	UUID string
}

func (s *Server) renderTemplate(name string, w http.ResponseWriter, data *templateData) {
	var buf strings.Builder
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set(headerContentType, contentTypeHTML+"; charset=utf-8")
	_, _ = io.WriteString(w, buf.String())
}

func negotiateContentType(r *http.Request) string {
	acceptHeader := r.Header.Get(headerAccept)
	if strings.Contains(acceptHeader, contentTypeHTML) {
		return contentTypeHTML
	}
	return contentTypePlain
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	contentType := negotiateContentType(r)
	switch contentType {
	case contentTypeHTML:
		s.renderTemplate("base", w, &templateData{})
	default:
		w.Header().Set(headerContentType, contentTypePlain)
		_, _ = fmt.Fprintln(w, "pastebin service - POST a 'blob' form field to create a paste")
	}
}

func (s *Server) handlePaste(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPasteSize)

	blob := r.FormValue(formFieldBlob)
	if len(blob) == 0 {
		http.Error(w, "Bad Request: empty paste", http.StatusBadRequest)
		return
	}

	pasteID := RandomString(pasteIDLength)

	// Retry on the extremely unlikely collision.
	for retries := 0; retries < maxCollisionRetries; retries++ {
		if _, found := s.store.Get(pasteID); !found {
			break
		}
		pasteID = RandomString(pasteIDLength)
	}
	if _, found := s.store.Get(pasteID); found {
		http.Error(w, "Internal Server Error: ID collision", http.StatusInternalServerError)
		return
	}
	s.store.Set(pasteID, blob, cache.DefaultExpiration)

	pastePath, err := url.Parse(fmt.Sprintf("./p/%s", pasteID))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resolvedURL := r.URL.ResolveReference(pastePath).String()
	contentType := negotiateContentType(r)

	switch contentType {
	case contentTypeHTML:
		http.Redirect(w, r, resolvedURL, http.StatusFound)
	default:
		_, _ = fmt.Fprint(w, r.Host+resolvedURL)
	}
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	pasteID := r.PathValue("uuid")
	if pasteID == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	rawBlob, ok := s.store.Get(pasteID)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	blob, ok := rawBlob.(string)
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	blob = strings.ReplaceAll(blob, "\t", "    ")

	contentType := negotiateContentType(r)
	switch contentType {
	case contentTypeHTML:
		s.renderTemplate("base", w, &templateData{
			Blob: blob,
			UUID: pasteID,
		})
	default:
		w.Header().Set(headerContentType, contentTypePlain)
		_, _ = fmt.Fprint(w, blob)
	}
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	pasteID := r.PathValue("uuid")
	if pasteID == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	_, ok := s.store.Get(pasteID)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	s.store.Delete(pasteID)
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "Deleted")
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	pasteID := r.PathValue("uuid")
	if pasteID == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	rawBlob, ok := s.store.Get(pasteID)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	blob, ok := rawBlob.(string)
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	content := strings.NewReader(blob)

	w.Header().Set(headerContentDisposition, "attachment; filename="+pasteID)
	w.Header().Set(headerContentType, "application/octet-stream")
	w.Header().Set(headerContentLength, strconv.FormatInt(content.Size(), 10))

	http.ServeContent(w, r, pasteID, time.Now(), content)
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats := struct {
		ItemCount int `json:"item_count"`
	}{
		ItemCount: s.store.ItemCount(),
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("error encoding stats: %v", err)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{Status: statusHealthy}); err != nil {
		log.Printf("error encoding health response: %v", err)
	}
}
