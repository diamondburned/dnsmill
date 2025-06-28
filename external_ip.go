package dnsmill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"
)

// ExternalIPProvider is the URL of the external IP provider.
const ExternalIPProvider = "https://ifconfig.me/ip"

// ResolveExternalIPs retrieves the external IP addresses of the machine.
func ResolveExternalIPs(ctx context.Context, flags HostAddressFlags) ([]net.IPAddr, error) {
	nets := []string{"tcp4", "tcp6"}
	switch {
	case flags.Has(HostAddressIPv4Only):
		nets = []string{"tcp4"}
	case flags.Has(HostAddressIPv6Only):
		nets = []string{"tcp6"}
	}

	addrs := make([]net.IPAddr, 0, len(nets))
	var errs []error

	for _, net := range nets {
		addr, err := resolveExternalIPForNetwork(ctx, net)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"failed to resolve external IP using %q: %w",
				net, err))
		}
		addrs = append(addrs, *addr)
	}

	if len(addrs) == 0 {
		if len(errs) > 0 {
			return nil, errors.Join(errs...)
		}
		return nil, fmt.Errorf("no external IP addresses resolved using %q", ExternalIPProvider)
	}

	return addrs, nil
}

// ResolveExternalIPsForLocalAddrs retrieves the external IP addresses for
// a list of local addresses.
func ResolveExternalIPsForLocalAddrs(ctx context.Context, localAddrs []net.IPAddr) ([]net.IPAddr, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resolvedAddrs := make([]net.IPAddr, 0, len(localAddrs))
	var resolveErrors []error

	for _, localAddr := range localAddrs {
		externalAddr, err := resolveExternalIPForLocalAddr(ctx, localAddr)
		if err != nil {
			resolveErrors = append(resolveErrors, fmt.Errorf(
				"failed to resolve external IP for local address %q: %w",
				localAddr, err))
			continue
		}
		resolvedAddrs = append(resolvedAddrs, *externalAddr)
	}

	slices.SortFunc(resolvedAddrs, func(a, b net.IPAddr) int {
		return strings.Compare(a.String(), b.String())
	})
	resolvedAddrs = slices.CompactFunc(resolvedAddrs, func(a, b net.IPAddr) bool {
		return a.String() == b.String()
	})

	if len(resolvedAddrs) == 0 {
		if len(resolveErrors) > 0 {
			return nil, errors.Join(resolveErrors...)
		}
		return nil, fmt.Errorf("no external IP addresses resolved for local addresses: %v", localAddrs)
	}

	return resolvedAddrs, nil
}

const dialTimeout = 5 * time.Second

func resolveExternalIPForLocalAddr(ctx context.Context, localAddr net.IPAddr) (*net.IPAddr, error) {
	dialer := &net.Dialer{
		Timeout: dialTimeout,
		LocalAddr: &net.TCPAddr{
			IP:   localAddr.IP,
			Zone: localAddr.Zone,
			Port: 0,
		},
	}
	httpTransport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	return resolveExternalIPForTransport(ctx, httpTransport)
}

func resolveExternalIPForNetwork(ctx context.Context, network string) (*net.IPAddr, error) {
	dialer := &net.Dialer{
		Timeout: dialTimeout,
	}
	httpTransport := &http.Transport{
		DialContext: func(ctx context.Context, _ string, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
	}
	return resolveExternalIPForTransport(ctx, httpTransport)
}

func resolveExternalIPForTransport(ctx context.Context, httpTransport *http.Transport) (*net.IPAddr, error) {
	httpClient := &http.Client{Transport: httpTransport}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ExternalIPProvider, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	respBodyStr := strings.TrimSpace(string(respBody))

	addr, err := net.ResolveIPAddr("ip", respBodyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve external IP %q: %w", respBodyStr, err)
	}

	return addr, nil
}
