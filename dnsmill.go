package dnsmill

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/libdns/libdns"
	"libdb.so/dnsmill/internal/cfgutil"
)

// Domain represents a domain name.
type Domain string

// Domains represents a list of domain names.
//
// When parsing, it may either parse a single domain name or a list of domain
// names.
type Domains []Domain

func (d *Domains) UnmarshalJSON(data []byte) error {
	var items []string
	if bytes.HasPrefix(data, []byte{'['}) {
		if err := json.Unmarshal(data, &items); err != nil {
			return fmt.Errorf("failed to parse Domains array: %w", err)
		}
	} else {
		var item string
		if err := json.Unmarshal(data, &item); err != nil {
			return fmt.Errorf("failed to parse Domains string: %w", err)
		}
		items = []string{item}
	}

	*d = make(Domains, len(items))
	for i, item := range items {
		(*d)[i] = Domain(item)
	}

	return nil
}

// Subdomain represents a subdomain of a root domain. It does not include the
// root domain itself. If "@", it represents the root domain.
type Subdomain string

// RootDomain represents the root domain as a special subdomain.
const RootDomain Subdomain = "@"

// SubdomainRecords maps subdomains to their DNS records.
type SubdomainRecords map[Subdomain]Records

// Convert converts the subdomain records into a list of [libdns.Record]s.
func (r SubdomainRecords) Convert(ctx context.Context) ([]libdns.Record, error) {
	var records []libdns.Record
	for subdomain, rec := range r {
		recs, err := rec.Convert(ctx, subdomain)
		if err != nil {
			return nil, fmt.Errorf("failed to convert records for %q: %w", subdomain, err)
		}
		records = append(records, recs...)
	}
	return records, nil
}

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
	CNAME string `json:"cname,omitempty"`
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
		if err := cfgutil.AssertSingleMapKey(data); err != nil {
			return err
		}

		type recordsAlias Records
		var records recordsAlias
		if err := json.Unmarshal(data, &records); err != nil {
			return fmt.Errorf("failed to parse records as Records: %w", err)
		}

		*r = Records(records)
	default:
		return errors.New("invalid JSON: expected string, array, or object")
	}

	return nil
}

// Convert converts the records assigned to the given subdomain into a list of
// [libdns.Record]s.
func (r *Records) Convert(ctx context.Context, subdomain Subdomain) ([]libdns.Record, error) {
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
				Name:  string(subdomain),
				Value: ip.String(),
			})
		}
	case r.CNAME != "":
		records = []libdns.Record{
			{
				Type:  "CNAME",
				Name:  string(subdomain),
				Value: r.CNAME,
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
