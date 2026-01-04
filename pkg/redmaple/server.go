package redmaple

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path"
	"time"
)

type Server struct {
	s      http.Server
	config Config
}

func NewServer(config Config) (*Server, error) {
	mux := http.NewServeMux()

	s := Server{
		s: http.Server{
			Addr:         fmt.Sprintf("0.0.0.0:%d", config.Port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler:      mux,
		},
		config: config,
	}

	s.LoadRoutes(mux)

	return &s, nil
}

func (s *Server) LoadRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", s.HandleIndex)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.config.StaticDir))))

	mux.Handle("GET /vendor/departure-mono/", http.StripPrefix("/vendor/departure-mono/", http.FileServer(http.Dir(s.config.VendorDir+"/departure-mono"))))
	mux.Handle("GET /vendor/htmx/", http.StripPrefix("/vendor/htmx/", http.FileServer(http.Dir(s.config.VendorDir+"/htmx"))))
	mux.Handle("GET /vendor/weather-icons/", http.StripPrefix("/vendor/weather-icons/", http.FileServer(http.Dir(s.config.VendorDir+"/weather-icons"))))
}

func (s *Server) Start() error {
	// start the HTTP server
	slog.Debug("listening", "addr", s.s.Addr)
	if err := s.s.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("http listen error", "error", err)
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) {
	s.s.Shutdown(ctx)
}

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "Index", struct{}{})
}

func (s *Server) loadTemplates(w http.ResponseWriter) *template.Template {
	plate, err := template.ParseGlob(path.Join(s.config.StaticDir, "pages/*.html"))
	if err != nil {
		slog.Error("template html parse failure", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}
	plate, err = plate.ParseGlob(path.Join(s.config.StaticDir, "partials/*.html"))
	if err != nil {
		slog.Error("template snippets parse failure", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}
	return plate
}

func (s *Server) executeTemplate(w http.ResponseWriter, name string, data any) {
	// TODO optimize template loading
	plate := s.loadTemplates(w)
	if plate == nil {
		return
	}
	if err := plate.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("template execution failure", "name", name, "error", err, "data", data)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
