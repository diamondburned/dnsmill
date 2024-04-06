package dnsmill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/invopop/yaml"
)

// Profile represents a DNS profile. It contains the configuration and DNS
// records for multiple domains.
//
// When parsing the JSON, if no "domains" field is present, then every field
// that is not "config" is considered a domain name and is filled into the
// "domains" field.
type Profile struct {
	// Config configures the behavior of the DNS tool.
	Config Config `json:"config,omitempty"`
	// Providers lists the DNS providers that are used in the profile.
	// Note that each provider will require its own secret environment variables
	// to be set.
	Providers map[string]ProviderConfig `json:"providers,omitempty"`
	// Domains maps domain names to their subdomains and DNS records.
	Domains map[Domain]SubdomainRecords `json:"domains,omitempty"`
}

// NewProfile creates a new empty profile with a default config.
func NewProfile() *Profile {
	return &Profile{
		Config: DefaultConfig(),
	}
}

func parseProfileAsYAML(r io.Reader) (*Profile, error) {
	// invopop/yaml does not support io.Reader.
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	p := NewProfile()
	if err := yaml.Unmarshal(b, p); err != nil {
		return nil, err
	}

	return p, nil
}

// ParseProfileAsYAML parses the profile from the YAML data.
func ParseProfileAsYAML(r io.Reader) (*Profile, error) {
	p, err := parseProfileAsYAML(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return p, p.Validate()
}

func parseProfileAsJSON(r io.Reader) (*Profile, error) {
	p := NewProfile()
	if err := json.NewDecoder(r).Decode(p); err != nil {
		return nil, err
	}
	return p, nil
}

// ParseProfileAsJSON parses the profile from the JSON data.
func ParseProfileAsJSON(r io.Reader) (*Profile, error) {
	p, err := parseProfileAsJSON(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return p, p.Validate()
}

// UnmarshalJSON unmarshals the JSON data into the profile.
func (p *Profile) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if _, ok := raw["config"]; ok {
		if err := json.Unmarshal(raw["config"], &p.Config); err != nil {
			return fmt.Errorf("failed to parse profile config JSON: %w", err)
		}
		delete(raw, "config")
	}

	if _, ok := raw["providers"]; ok {
		if err := json.Unmarshal(raw["providers"], &p.Providers); err != nil {
			return fmt.Errorf("failed to parse providers JSON: %w", err)
		}
		delete(raw, "providers")
	}

	if _, ok := raw["domains"]; ok {
		if err := json.Unmarshal(raw["domains"], &p.Domains); err != nil {
			return fmt.Errorf("failed to parse domains JSON in field: %w", err)
		}
		delete(raw, "domains")
		if len(raw) > 0 {
			return fmt.Errorf("unexpected fields in profile JSON: %v", raw)
		}
		// stop parsing
		return nil
	}

	p.Domains = make(map[Domain]SubdomainRecords, len(raw))
	for domain, records := range raw {
		var v SubdomainRecords
		if err := json.Unmarshal(records, &v); err != nil {
			return fmt.Errorf("failed to parse domain %s JSON: %w", domain, err)
		}
		p.Domains[Domain(domain)] = v
	}

	return nil
}

// Validate validates the profile.
func (p *Profile) Validate() error {
	// Ensure that every domain is known by exactly one provider.
	for domain := range p.Domains {
		name, err := p.findDomainProviderName(domain)
		if err != nil {
			return err
		}
		if _, err = getProvider(name); err != nil {
			return fmt.Errorf("domain %q: %w", domain, err)
		}
	}

	return nil
}

func (p *Profile) findDomainProviderName(domain Domain) (string, error) {
	for name, config := range p.Providers {
		if slices.Contains(config.Domains, domain) {
			return name, nil
		}
	}

	return "", fmt.Errorf("domain %q is not managed by any provider", domain)
}

// Apply applies the profile to the DNS providers.
//
// If dryRun is true, it will convert the profile to libdns records and log them
// without applying them to the providers. This is useful for debugging and
// testing the profile without affecting the DNS records.
//
// If an error occurs, it will finish applying the profile to other zones before
// returning the error. This way, the profile is applied as much as possible
// before failing. Multiple errors will be joined using [errors.Join].
func (p *Profile) Apply(ctx context.Context, logger *slog.Logger, dryRun bool) error {
	// Ensure that the profile is valid before applying it.
	// This way, any config errors are caught early.
	if err := p.Validate(); err != nil {
		return fmt.Errorf("failed to validate profile: %w", err)
	}

	providers := make(map[string]Provider, len(p.Providers))
	for name := range p.Providers {
		factory, err := getProvider(name)
		if err != nil {
			return fmt.Errorf("failed to get provider %q: %w", name, err)
		}
		p, err := factory.New(ctx)
		if err != nil {
			return fmt.Errorf("failed to create provider %q: %w", name, err)
		}
		providers[name] = p
	}

	apply := func(domain Domain, records SubdomainRecords, logger *slog.Logger) error {
		providerName, _ := p.findDomainProviderName(domain)
		provider := providers[providerName]

		libdnsRecords, err := records.Convert(ctx)
		if err != nil {
			return fmt.Errorf("failed to convert records for %q: %w", domain, err)
		}

		for _, record := range libdnsRecords {
			logger.Info(
				"applying fresh libdns record",
				"provider", providerName,
				"record.type", record.Type,
				"record.name", record.Name,
				"record.value", record.Value)
		}

		if dryRun {
			logger.Debug("dry run enabled, skipping application")
			return nil
		}

		switch p.Config.DuplicatePolicy {
		case ErrorOnDuplicate:
			libdnsRecords, err = provider.AppendRecords(ctx, string(domain), libdnsRecords)
		case OverwriteDuplicate:
			libdnsRecords, err = provider.SetRecords(ctx, string(domain), libdnsRecords)
		default:
			panic("unreachable")
		}

		if err != nil {
			return fmt.Errorf("failed to apply records for %q: %w", domain, err)
		}

		for _, record := range libdnsRecords {
			logger.Info(
				"applied libdns record",
				"provider", providerName,
				"record.id", record.ID,
				"record.type", record.Type,
				"record.name", record.Name,
				"record.value", record.Value)
		}

		return nil
	}

	var errs []error

	// TODO: parallelize
	for domain, records := range p.Domains {
		logger := logger.With("domain", domain)
		if err := apply(domain, records, logger); err != nil {
			logger.Error(
				"cannot apply domain",
				"err", err)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
