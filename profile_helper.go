package dnsmill

import (
	"fmt"
	"slices"
)

type profileMapped struct {
	*Profile
	rootDomains []mappedRootDomain
}

type mappedRootDomain struct {
	RootDomain   Domain
	Subdomains   DomainRecords
	ProviderName string
}

func mapRootDomains(p *Profile) ([]mappedRootDomain, error) {
	var rootDomains []mappedRootDomain
	for providerName, providerConfig := range p.Providers {
		for _, rootDomain := range providerConfig.Zones {
			if slices.ContainsFunc(rootDomains, func(p mappedRootDomain) bool {
				return p.RootDomain == rootDomain
			}) {
				return nil, fmt.Errorf("domain %q is already managed by another provider", rootDomain)
			}
		}

		rootDomains = slices.Grow(rootDomains, len(providerConfig.Zones))
		for _, rootDomain := range providerConfig.Zones {
			rootDomains = append(rootDomains, mappedRootDomain{
				RootDomain:   rootDomain,
				Subdomains:   DomainRecords{},
				ProviderName: providerName,
			})
		}
	}

	for domain, records := range p.Records {
		rootDomainIx := slices.IndexFunc(rootDomains, func(d mappedRootDomain) bool {
			return domain.IsSubdomainOf(d.RootDomain)
		})
		if rootDomainIx == -1 {
			return nil, fmt.Errorf("domain %q is not managed by any provider", domain)
		}

		rootDomain := &rootDomains[rootDomainIx]
		rootDomain.Subdomains[domain] = records
	}

	return rootDomains, nil
}
