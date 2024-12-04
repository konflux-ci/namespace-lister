package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func main() {
	l := buildLogger()
	if err := run(l); err != nil {
		l.Error("error running the server", "error", err)
		os.Exit(1)
	}
}

func run(l *slog.Logger) error {
	log.SetLogger(logr.FromSlogHandler(l.Handler()))

	// get config
	cfg := ctrl.GetConfigOrDie()

	// build the request authenticator
	ar, err := NewAuthenticator(AuthenticatorOptions{
		Config: cfg,
		Header: GetUsernameHeaderFromEnv(),
	})
	if err != nil {
		return err
	}

	// setup context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ctx = setLoggerIntoContext(ctx, l)

	// create cache
	l.Info("creating cache")
	cache, err := BuildAndStartCache(ctx, cfg)
	if err != nil {
		return err
	}

	// create the authorizer and the namespace lister
	auth := NewAuthorizer(ctx, cache)
	nsl := NewNamespaceLister(cache, auth)

	// build http server
	l.Info("building server")
	s := NewServer(l, ar, nsl)

	// start the server
	return s.Start(ctx)
}
