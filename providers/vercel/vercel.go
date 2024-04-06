// Package vercel provides the Vercel DNS provider for dnsmill. It requires
// the following environment variables to be set:
//   - `VERCEL_AUTH_TOKEN`: The authentication token for the Vercel account
//     (see https://vercel.com/docs/api#api-basics/authentication).
package vercel

import (
	"context"
	"os"

	"github.com/libdns/vercel"
	"libdb.so/dnsmill"
	"libdb.so/dnsmill/internal/envutil"
)

func init() {
	dnsmill.RegisterProvider(dnsmill.ProviderFactory{
		New:    newProvider,
		Name:   "vercel",
		DocURL: "https://pkg.go.dev/libdb.so/dnsmill/providers/vercel",
	})
}

func newProvider(context.Context) (dnsmill.Provider, error) {
	if err := envutil.AssertEnv("VERCEL_AUTH_TOKEN"); err != nil {
		return nil, err
	}

	return &vercel.Provider{
		AuthAPIToken: os.Getenv("VERCEL_AUTH_TOKEN"),
	}, nil
}
