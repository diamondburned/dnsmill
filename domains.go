package dnsmill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/libdns/libdns"
)

// Domain represents a domain name, such as "example.com" or "app.example.com".
// Not all instances of Domain types can accept subdomains.
type Domain string

// SubdomainOf returns the subdomain of the domain if it is a subdomain of the
// given root domain. It returns ("", true) if d == rootDomain.
func (d Domain) SubdomainOf(rootDomain Domain) (string, bool) {
	if d == rootDomain {
		return "", true
	}
	if strings.HasSuffix(string(d), "."+string(rootDomain)) {
		subdomain := string(d)
		subdomain = strings.TrimSuffix(subdomain, "."+string(rootDomain))
		subdomain = strings.TrimSuffix(subdomain, ".")
		return subdomain, true
	}
	return "", false
}

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

// DomainRecords maps subdomains to their DNS records.
type DomainRecords map[Domain]Records

// Convert converts the subdomain records into a list of [libdns.Record]s.
func (r DomainRecords) Convert(ctx context.Context, rootDomain Domain) ([]libdns.Record, error) {
	records := make([]libdns.Record, 0, len(r))

	for domain, rec := range r {
		subdomain, ok := domain.SubdomainOf(rootDomain)
		if domain != rootDomain && !ok {
			return nil, fmt.Errorf("%q is not a subdomain of %q", domain, rootDomain)
		}

		recs, err := rec.Convert(ctx, subdomain)
		if err != nil {
			return nil, fmt.Errorf("failed to convert records for %q: %w", domain, err)
		}

		records = append(records, recs...)
	}

	return records, nil
}
