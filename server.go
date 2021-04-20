package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	// Logging
	"github.com/unrolled/logger"

	// Stats/Metrics
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/thoas/stats"

	rice "github.com/GeertJohan/go.rice"
	"github.com/julienschmidt/httprouter"
	"github.com/patrickmn/go-cache"
	"github.com/timewasted/go-accept-headers"
)

// AcceptedTypes ...
var AcceptedTypes = []string{
	"text/html",
	"text/plain",
}

// Counters ...
type Counters struct {
	r metrics.Registry
}

func NewCounters() *Counters {
	counters := &Counters{
		r: metrics.NewRegistry(),
	}
	return counters
}

func (c *Counters) Inc(name string) {
	metrics.GetOrRegisterCounter(name, c.r).Inc(1)
}

func (c *Counters) Dec(name string) {
	metrics.GetOrRegisterCounter(name, c.r).Dec(1)
}

func (c *Counters) IncBy(name string, n int64) {
	metrics.GetOrRegisterCounter(name, c.r).Inc(n)
}

func (c *Counters) DecBy(name string, n int64) {
	metrics.GetOrRegisterCounter(name, c.r).Dec(n)
}

// Server ...
type Server struct {
	bind      string
	config    Config
	store     *cache.Cache
	templates *Templates
	router    *httprouter.Router

	// Logger
	logger *logger.Logger

	// Stats/Metrics
	counters *Counters
	stats    *stats.Stats
}

func (s *Server) render(name string, w http.ResponseWriter, ctx interface{}) {
	buf, err := s.templates.Exec(name, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// IndexHandler ...
func (s *Server) IndexHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		s.counters.Inc("n_index")

		accepts, err := accept.Negotiate(
			r.Header.Get("Accept"), AcceptedTypes...,
		)
		if err != nil {
			log.Printf("error negotiating: %s", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		switch accepts {
		case "text/html":
			s.render("index", w, nil)
		case "text/plain":
		default:
		}
	}
}

// PasteHandler ...
func (s *Server) PasteHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.counters.Inc("n_paste")

		accepts, err := accept.Negotiate(
			r.Header.Get("Accept"), AcceptedTypes...,
		)
		if err != nil {
			log.Printf("error negotiating: %s", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		blob := r.FormValue("blob")

		if len(blob) == 0 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		uuid := RandomString(8)
		s.store.Set(uuid, blob, cache.DefaultExpiration)

		u, err := url.Parse(fmt.Sprintf("./p/%s", uuid))
		if err != nil {
			http.Error(w, "Internal Error", http.StatusInternalServerError)
		}
		switch accepts {
		case "text/html":
			http.Redirect(w, r, r.URL.ResolveReference(u).String(), http.StatusFound)
		case "text/plain":
			fallthrough
		default:
			w.Write([]byte(r.Host + r.URL.ResolveReference(u).String()))
		}
	}
}

// DownloadHandler ...
func (s *Server) DownloadHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.counters.Inc("n_download")

		uuid := p.ByName("uuid")
		if uuid == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		blob, ok := s.store.Get(uuid)
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		content := strings.NewReader(blob.(string))

		w.Header().Set("Content-Disposition", "attachment; filename="+uuid)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.FormatInt(content.Size(), 10))

		http.ServeContent(w, r, uuid, time.Now(), content)
	}
}

// DeleteHandler
func (s *Server) DeleteHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.counters.Inc("n_delete")
		_, err := accept.Negotiate(
			r.Header.Get("Accept"), AcceptedTypes...,
		)
		if err != nil {
			log.Printf("error negotiating: %s", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		uuid := p.ByName("uuid")
		if uuid == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		_, ok := s.store.Get(uuid)
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		s.store.Delete(uuid)
		http.Error(w, "Deleted", http.StatusOK)
	}
}

// ViewHandler ...
func (s *Server) ViewHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.counters.Inc("n_view")

		accepts, err := accept.Negotiate(
			r.Header.Get("Accept"), AcceptedTypes...,
		)
		if err != nil {
			log.Printf("error negotiating: %s", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		uuid := p.ByName("uuid")
		if uuid == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		rawBlob, ok := s.store.Get(uuid)
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		blob := rawBlob.(string)
		blob = strings.ReplaceAll(blob, "\t", "    ")

		switch accepts {
		case "text/html":
			s.render(
				"view", w,
				struct {
					Blob string
					UUID string
				}{
					Blob: blob,
					UUID: uuid,
				},
			)
		case "text/plain":
			fallthrough
		default:
			w.Write([]byte(blob))
		}
	}
}

// StatsHandler ...
func (s *Server) StatsHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		bs, err := json.Marshal(s.stats.Data())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(bs)
	}
}

// ListenAndServe ...
func (s *Server) ListenAndServe() {
	log.Fatal(
		http.ListenAndServe(
			s.bind,
			s.logger.Handler(
				s.stats.Handler(s.router),
			),
		),
	)
}

func (s *Server) initRoutes() {
	s.router.Handler("GET", "/debug/metrics", exp.ExpHandler(s.counters.r))
	s.router.GET("/debug/stats", s.StatsHandler())

	s.router.ServeFiles(
		"/css/*filepath",
		rice.MustFindBox("static/css").HTTPBox(),
	)

	s.router.GET("/", s.IndexHandler())
	s.router.POST("/", s.PasteHandler())
	s.router.GET("/download/:uuid", s.DownloadHandler())
	s.router.GET("/p/:uuid", s.ViewHandler())
	// Enable DELETE from curl/wget/cli
	s.router.DELETE("/p/:uuid", s.DeleteHandler())
	// Add alternate path since form actions don't support method=DELETE
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/DELETE
	s.router.POST("/delete/:uuid", s.DeleteHandler())
}

// NewServer ...
func NewServer(bind string, config Config) *Server {
	server := &Server{
		bind:      bind,
		config:    config,
		router:    httprouter.New(),
		store:     cache.New(cfg.expiry, cfg.expiry*2),
		templates: NewTemplates("base"),

		// Logger
		logger: logger.New(logger.Options{
			Prefix:               "pastebin",
			RemoteAddressHeaders: []string{"X-Forwarded-For"},
			OutputFlags:          log.LstdFlags,
		}),

		// Stats/Metrics
		counters: NewCounters(),
		stats:    stats.New(),
	}

	// Templates
	box := rice.MustFindBox("templates")

	indexTemplate := template.New("index")
	template.Must(indexTemplate.Parse(box.MustString("index.html")))
	template.Must(indexTemplate.Parse(box.MustString("base.html")))

	viewTemplate := template.New("view")
	template.Must(viewTemplate.Parse(box.MustString("view.html")))
	template.Must(viewTemplate.Parse(box.MustString("base.html")))

	server.templates.Add("index", indexTemplate)
	server.templates.Add("view", viewTemplate)

	server.initRoutes()

	return server
}
