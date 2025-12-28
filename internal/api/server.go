package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server is the HTTP API server for runtime control
type Server struct {
	server   *http.Server
	handlers *Handlers
	logger   *slog.Logger
}

// ServerConfig holds configuration for the API server
type ServerConfig struct {
	Port   int
	Logger *slog.Logger
}

// NewServer creates a new API server
func NewServer(cfg ServerConfig, manager TargetManager) *Server {
	handlers := NewHandlers(manager, cfg.Logger)

	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("GET /api/targets", handlers.ListTargets)
	mux.HandleFunc("POST /api/targets", handlers.AddTarget)
	mux.HandleFunc("GET /api/targets/{name}", handlers.GetTarget)
	mux.HandleFunc("DELETE /api/targets/{name}", handlers.RemoveTarget)
	mux.HandleFunc("POST /api/targets/{name}/start", handlers.StartTarget)
	mux.HandleFunc("POST /api/targets/{name}/stop", handlers.StopTarget)
	mux.HandleFunc("GET /api/targets/{name}/results", handlers.GetTargetResults)
	mux.HandleFunc("GET /api/status", handlers.GetStatus)
	mux.HandleFunc("GET /api/health", handlers.HealthCheck)

	// Wrap with middleware
	handler := loggingMiddleware(cfg.Logger, recoveryMiddleware(jsonContentTypeMiddleware(mux)))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		server:   server,
		handlers: handlers,
		logger:   cfg.Logger,
	}
}

// Start starts the API server (blocking)
func (s *Server) Start() error {
	s.logger.Info("starting API server", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("API server failed: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down API server")
	return s.server.Shutdown(ctx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.server.Addr
}

// jsonContentTypeMiddleware sets JSON content type for API responses
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics and returns 500 errors
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal server error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		logger.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start).String(),
			"remote_addr", r.RemoteAddr)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
