package suite

import (
	"cmp"
	"os"
)

const EnvKonfluxAddress string = "KONFLUX_ADDRESS"

func EnvKonfluxAddressOrDefault(address string) string {
	return cmp.Or(os.Getenv(EnvKonfluxAddress), address)
}
