// Package namecheap provides the Namecheap DNS provider for dnsmill. It requires
// the following environment variables to be set:
//   - `NAMECHEAP_API_USER`: The API user for the Namecheap account, usually
//     your username.
//   - `NAMECHEAP_API_KEY`: The API key for the Namecheap account.
//     See https://www.namecheap.com/support/api/intro/ for more information.
//   - `NAMECHEAP_API_ENDPOINT`: The API endpoint for the Namecheap account.
//     If testing, you may use "sandbox" instead of the production endpoint.
//   - `NAMECHEAP_CLIENT_IP`: The client IP address for the Namecheap account.
//     If not set, the machine's public IP address is used.
//   - `NAMECHEAP_IP_RESOLVER`: The resolver to use for resolving the client's
//     public IP address. If not set, ifconfig.me is used.
package namecheap

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/libdns/namecheap"
	"libdb.so/dnsmill"
	"libdb.so/dnsmill/internal/envutil"
)

func init() {
	dnsmill.RegisterProvider(dnsmill.ProviderFactory{
		New:    newProvider,
		Name:   "namecheap",
		DocURL: "https://pkg.go.dev/libdb.so/dnsmill/providers/namecheap",
	})
}

func newProvider(ctx context.Context) (dnsmill.Provider, error) {
	if err := envutil.AssertEnv(
		"NAMECHEAP_API_USER",
		"NAMECHEAP_API_KEY",
		"NAMECHEAP_API_ENDPOINT",
	); err != nil {
		return nil, err
	}

	clientIP := os.Getenv("NAMECHEAP_CLIENT_IP")
	if clientIP == "" {
		ipResolver := os.Getenv("NAMECHEAP_IP_RESOLVER")
		if ipResolver != "" {
			ipResolver = "https://ifconfig.me"
		}
		ip, err := resolveExternalIP(ctx, ipResolver)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve external IP: %w", err)
		}
		clientIP = ip
	}

	return &namecheap.Provider{
		APIKey:      os.Getenv("NAMECHEAP_API_KEY"),
		User:        os.Getenv("NAMECHEAP_API_USER"),
		APIEndpoint: os.Getenv("NAMECHEAP_API_ENDPOINT"),
		ClientIP:    clientIP,
	}, nil
}

func resolveExternalIP(ctx context.Context, ipResolver string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", ipResolver, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	var s strings.Builder
	if _, err := io.Copy(&s, resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned non-200 status %d: %v", resp.StatusCode, s.String())
	}

	if ip := net.ParseIP(s.String()); ip == nil {
		return "", fmt.Errorf("server returned unexpected data: %q", s.String())
	}

	return s.String(), nil
}
