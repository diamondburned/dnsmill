// Package netlify provides the Netlify DNS provider for dnsmill. It requires
// the following environment variables to be set:
//   - `NETLIFY_TOKEN`: The Personal Access Token for the Netlify account.
package netlify

import (
	"context"
	"os"

	"github.com/libdns/netlify"
	"libdb.so/dnsmill"
	"libdb.so/dnsmill/internal/envutil"
)

func init() {
	dnsmill.RegisterProvider(dnsmill.ProviderFactory{
		New:    newProvider,
		Name:   "netlify",
		DocURL: "https://pkg.go.dev/libdb.so/dnsmill/providers/netlify",
	})
}

func newProvider(context.Context) (dnsmill.Provider, error) {
	if err := envutil.AssertEnv("NETLIFY_TOKEN"); err != nil {
		return nil, err
	}

	return &netlify.Provider{
		PersonalAccessToken: os.Getenv("NETLIFY_TOKEN"),
	}, nil
}
