package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	patternGetNamespaces string = "GET /api/v1/namespaces"
)

type NamespaceListerServer struct {
	*http.Server

	logger *slog.Logger
}

func addLogMiddleware(l *slog.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Info("received request", "request", r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func NewServer(l *slog.Logger, lister NamespaceLister, userHeader string) *NamespaceListerServer {
	// configure the server
	h := http.NewServeMux()
	h.Handle(patternGetNamespaces, addLogMiddleware(l, NewListNamespacesHandler(l, lister, userHeader)))
	return &NamespaceListerServer{
		Server: &http.Server{
			Addr:              getAddress(),
			Handler:           h,
			ReadHeaderTimeout: 3 * time.Second,
		},
		logger: l,
	}
}

func (s *NamespaceListerServer) Start(ctx context.Context) error {
	// HTTP Server graceful shutdown
	go func() {
		<-ctx.Done()

		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		//nolint:contextcheck
		if err := s.Shutdown(sctx); err != nil {
			s.logger.Error("error gracefully shutting down the HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	// start server
	s.logger.Info("serving...")
	return s.ListenAndServe()
}
