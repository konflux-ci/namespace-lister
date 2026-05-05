package main

import (
	"cmp"
	"os"

	"github.com/konflux-ci/namespace-lister/internal/envconfig"
)

func getAddress() string {
	return cmp.Or(os.Getenv(envconfig.EnvAddress), envconfig.DefaultAddr)
}
