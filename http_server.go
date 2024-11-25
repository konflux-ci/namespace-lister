package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"k8s.io/apiserver/pkg/authentication/authenticator"
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

func addAuthnMiddleware(ar authenticator.Request, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs, ok, err := ar.AuthenticateRequest(r)

		switch {
		case err != nil: // error contacting the APIServer for authenticating the request
			w.WriteHeader(http.StatusUnauthorized)
			l := getLoggerFromContext(r.Context())
			l.Error("error authenticating request", "error", err, "request-headers", r.Header)
			return

		case !ok: // request could not be authenticated
			w.WriteHeader(http.StatusUnauthorized)
			return

		default: // request is authenticated
			// Inject authentication details into request context
			ctx := r.Context()
			authCtx := context.WithValue(ctx, ContextKeyUserDetails, rs)

			// serve next request
			next.ServeHTTP(w, r.WithContext(authCtx))
		}
	}
}

func NewServer(l *slog.Logger, ar authenticator.Request, lister NamespaceLister) *NamespaceListerServer {
	// configure the server
	h := http.NewServeMux()
	h.Handle(patternGetNamespaces,
		addInjectLoggerMiddleware(l,
			addLogRequestMiddleware(
				addAuthnMiddleware(ar,
					NewListNamespacesHandler(lister)))))

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
