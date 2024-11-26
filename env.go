package main

import (
	"cmp"
	"os"
)

func getAddress() string {
	return cmp.Or(os.Getenv(EnvAddress), DefaultAddr)
}
