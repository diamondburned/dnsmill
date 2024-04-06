// Package porkbun provides the Porkbun DNS provider for dnsmill. It requires
// the following environment variables to be set:
//   - `PORKBUN_API_KEY`: The API key for the Porkbun account.
//   - `PORKBUN_SECRET_KEY`: The secret key for the Porkbun account.
package porkbun

import (
	"context"
	"os"

	"github.com/libdns/porkbun"
	"libdb.so/dnsmill"
	"libdb.so/dnsmill/internal/envutil"
)

func init() {
	dnsmill.RegisterProvider(dnsmill.ProviderFactory{
		New:    newProvider,
		Name:   "porkbun",
		DocURL: "https://pkg.go.dev/libdb.so/dnsmill/providers/porkbun",
	})
}

func newProvider(context.Context) (dnsmill.Provider, error) {
	if err := envutil.AssertEnv("PORKBUN_API_KEY", "PORKBUN_SECRET_KEY"); err != nil {
		return nil, err
	}

	return &porkbun.Provider{
		APIKey:       os.Getenv("PORKBUN_API_KEY"),
		APISecretKey: os.Getenv("PORKBUN_SECRET_KEY"),
	}, nil
}
