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
}

func addInjectLoggerMiddleware(l *slog.Logger, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := setLoggerIntoContext(r.Context(), l)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func addLogRequestMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := getLoggerFromContext(r.Context())
		l.Info("received request", "request", r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func NewServer(l *slog.Logger, lister NamespaceLister, userHeader string) *NamespaceListerServer {
	// configure the server
	h := http.NewServeMux()
	h.Handle(patternGetNamespaces,
		addInjectLoggerMiddleware(l,
			addLogRequestMiddleware(
				NewListNamespacesHandler(lister, userHeader))))
	return &NamespaceListerServer{
		Server: &http.Server{
			Addr:              getAddress(),
			Handler:           h,
			ReadHeaderTimeout: 3 * time.Second,
		},
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
			getLoggerFromContext(ctx).Error("error gracefully shutting down the HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	// start server
	return s.ListenAndServe()
}
