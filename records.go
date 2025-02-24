package dnsmill

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/libdns/libdns"
)

// Records represents a list of DNS records.
//
// When parsing, it may either parse:
//   - A single IPv4 or IPv6 address, which is parsed as a single host address.
//   - A list of IPv4 or IPv6 addresses and hostnames, which is parsed as a list
//     of host addresses.
//   - A struct with exactly a single field representing the type of record
//     corresponding to its value.
type Records struct {
	// Hosts indirectly represents A and AAAA records. The host addresses are
	// resolved into a list of IP addresses, with IPv4 addresses being handled
	// as A records and IPv6 addresses being handled as AAAA records.
	Hosts *HostAddresses `json:"hosts,omitempty"`

	// CNAME represents a single CNAME record.
	CNAME *string `json:"cname,omitempty"`
}

func (r *Records) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("invalid JSON: empty object")
	}

	switch data[0] {
	case '"', '[':
		// Parse the string-or-array as a HostAddresses instance.
		var hosts HostAddresses
		if err := json.Unmarshal(data, &hosts); err != nil {
			return fmt.Errorf("failed to parse records as HostAddresses: %w", err)
		}

		*r = Records{Hosts: &hosts}

	case '{':
		type recordsAlias Records
		var records recordsAlias
		if err := json.Unmarshal(data, &records); err != nil {
			return fmt.Errorf("failed to parse records as Records: %w", err)
		}

		if records.Hosts != nil && records.CNAME != nil {
			return errors.New("invalid JSON: expected either hosts or cname field, not both")
		}

		*r = Records(records)
	default:
		return errors.New("invalid JSON: expected string, array, or object")
	}

	return nil
}

// Convert converts the records assigned to the given subdomain into a list of
// [libdns.Record]s.
func (r *Records) Convert(ctx context.Context, subdomain string) ([]libdns.Record, error) {
	if subdomain == "" {
		subdomain = "@"
	}

	var records []libdns.Record

	switch {
	case r.Hosts != nil:
		ips, err := r.Hosts.ResolveIPs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve hosts: %w", err)
		}
		for _, ip := range ips {
			t := "AAAA"
			if v4 := ip.To4(); v4 != nil {
				t = "A"
				ip = v4
			}
			records = append(records, libdns.Record{
				Type:  t,
				Name:  subdomain,
				Value: ip.String(),
			})
		}
	case r.CNAME != nil:
		records = []libdns.Record{
			{
				Type:  "CNAME",
				Name:  subdomain,
				Value: *r.CNAME,
			},
		}
	}

	return records, nil
}

// HostAddresses represents a list of IP addresses and hostnames.
//
// When parsing, it may either parse a single host address or a list of host
// addresses.
type HostAddresses []string

func (a *HostAddresses) UnmarshalJSON(data []byte) error {
	var items []string
	if bytes.HasPrefix(data, []byte{'['}) {
		if err := json.Unmarshal(data, &items); err != nil {
			return fmt.Errorf("failed to parse HostAddresses array: %w", err)
		}
	} else {
		var item string
		if err := json.Unmarshal(data, &item); err != nil {
			return fmt.Errorf("failed to parse HostAddresses string: %w", err)
		}
		items = []string{item}
	}

	*a = items
	return nil
}

// ResolveIPs resolves the IP addresses if they are hostnames and returns the IP
// addresses. This function is not cached. [net.DefaultResolver] is used to
// resolve the hostnames.
func (a *HostAddresses) ResolveIPs(ctx context.Context) ([]net.IP, error) {
	ips := make([]net.IP, 0, len(*a))
	for _, addr := range *a {
		if ip := net.ParseIP(addr); ip != nil {
			ips = append(ips, ip)
			continue
		}

		resolved, err := net.DefaultResolver.LookupIPAddr(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %q: %w", addr, err)
		}
		if len(resolved) == 0 {
			return nil, fmt.Errorf("no IP addresses found for %q", addr)
		}

		for _, addr := range resolved {
			ips = append(ips, addr.IP)
		}
	}

	return ips, nil
}
