package dnsmill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/libdns/libdns"
)

// ProviderConfig describes the configuration for a DNS provider.
//
// When parsing as JSON, it can be parsed as a list of domains or the actual
// [ProviderConfig] instance itself.
type ProviderConfig struct {
	// Domains lists the domains that are managed by the provider.
	Domains Domains `json:"domains"`
}

func (c *ProviderConfig) UnmarshalJSON(data []byte) error {
	switch {
	case bytes.HasPrefix(data, []byte{'['}):
		var domains Domains
		if err := json.Unmarshal(data, &domains); err != nil {
			return fmt.Errorf("failed to parse ProviderConfig domains: %w", err)
		}
		*c = ProviderConfig{Domains: domains}
		return nil
	default:
		type alias ProviderConfig
		var cfg alias
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse ProviderConfig: %w", err)
		}
		*c = ProviderConfig(cfg)
		return nil
	}
}

// ProviderFactory allows a DNS provider to be created.
// The factory has metadata associated with it.
type ProviderFactory struct {
	// New creates a new DNS provider.
	New func(ctx context.Context) (Provider, error)
	// Name is the name of the DNS provider.
	Name string
	// DocURL is the URL to the documentation of the DNS provider.
	DocURL string
}

// Provider describes a DNS provider. It must be safe for concurrent use.
type Provider interface {
	libdns.RecordAppender // policy = error
	libdns.RecordSetter   // policy = overwrite
}

var providerRegistry = map[string]ProviderFactory{}

// RegisterProvider registers a DNS provider into the global registry.
func RegisterProvider(p ProviderFactory) {
	if _, ok := providerRegistry[p.Name]; ok {
		panic(fmt.Sprintf("provider %q already registered", p.Name))
	}
	providerRegistry[p.Name] = p
}

// ListProviders returns the list of registered DNS providers.
func ListProviders() []ProviderFactory {
	factories := make([]ProviderFactory, 0, len(providerRegistry))
	for _, f := range providerRegistry {
		factories = append(factories, f)
	}
	return factories
}

func getProvider(name string) (ProviderFactory, error) {
	p, ok := providerRegistry[name]
	if !ok {
		return ProviderFactory{}, fmt.Errorf("unknown provider: %q", name)
	}
	return p, nil
}
