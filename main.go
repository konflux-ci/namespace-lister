package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-logr/logr"

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

	// setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create cache
	l.Info("creating cache")
	cache, err := BuildAndStartCache(ctx)
	if err != nil {
		return err
	}

	// create the authorizer and the namespace lister
	auth := NewAuthorizer(ctx, cache, l)
	nsl := NewNamespaceLister(cache, auth, l)

	// build http server
	l.Info("building server")
	userHeader := getHeaderUsername()
	s := NewServer(l, nsl, userHeader)

	// start the server
	l.Info("serving...")
	return s.Start(ctx)
}
