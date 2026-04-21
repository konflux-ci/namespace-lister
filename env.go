package main

import (
	"cmp"
	"os"

	"github.com/konflux-ci/namespace-lister/internal/constant"
)

func getAddress() string {
	return cmp.Or(os.Getenv(constant.EnvAddress), constant.DefaultAddr)
}
