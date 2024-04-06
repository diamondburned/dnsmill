// Package cloudflare provides the Cloudflare DNS provider for dnsmill. It requires
// the following environment variables to be set:
//   - `CLOUDFLARE_API_TOKEN`: The API token for the Cloudflare account. Make
//     sure to use a scoped API token, not a global API key. It will need two
//     permissions: Zone-Zone-Read and Zone-DNS-Edit.
package cloudflare

import (
	"context"
	"os"

	"github.com/libdns/cloudflare"
	"libdb.so/dnsmill"
	"libdb.so/dnsmill/internal/envutil"
)

func init() {
	dnsmill.RegisterProvider(dnsmill.ProviderFactory{
		New:    newProvider,
		Name:   "cloudflare",
		DocURL: "https://pkg.go.dev/libdb.so/dnsmill/providers/cloudflare",
	})
}

func newProvider(context.Context) (dnsmill.Provider, error) {
	if err := envutil.AssertEnv("CLOUDFLARE_API_TOKEN"); err != nil {
		return nil, err
	}

	return &cloudflare.Provider{
		APIToken: os.Getenv("CLOUDFLARE_API_TOKEN"),
	}, nil
}
