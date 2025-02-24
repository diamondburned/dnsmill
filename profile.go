package dnsmill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

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
	// Records is a list of each domains' DNS records.
	Records DomainRecords `json:"records,omitempty"`
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

	if _, ok := raw["records"]; ok {
		if err := json.Unmarshal(raw["records"], &p.Records); err != nil {
			return fmt.Errorf("failed to parse records JSON in field: %w", err)
		}
		delete(raw, "records")
		if len(raw) > 0 {
			return fmt.Errorf("unexpected fields in profile JSON: %v", raw)
		}
		// stop parsing
		return nil
	}

	p.Records = make(DomainRecords, len(raw))
	for domain, records := range raw {
		var v Records
		if err := json.Unmarshal(records, &v); err != nil {
			return fmt.Errorf("failed to parse extra records %s JSON: %w", domain, err)
		}
		p.Records[Domain(domain)] = v
	}

	return nil
}

// Validate validates the profile.
func (p *Profile) Validate() error {
	_, err := mapRootDomains(p)
	if err != nil {
		return err
	}

	return nil
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

	apply := func(root mappedRootDomain, logger *slog.Logger) error {
		libdnsRecords, err := root.Subdomains.Convert(ctx, root.RootDomain)
		if err != nil {
			return fmt.Errorf("failed to convert records for %q: %w", root.RootDomain, err)
		}

		for _, record := range libdnsRecords {
			logger.Info(
				"applying fresh libdns record",
				"record.type", record.Type,
				"record.name", record.Name,
				"record.value", record.Value)
		}

		if dryRun {
			logger.Debug("dry run enabled, skipping application")
			return nil
		}

		provider := providers[root.ProviderName]
		switch p.Config.DuplicatePolicy {
		case ErrorOnDuplicate:
			libdnsRecords, err = provider.AppendRecords(ctx, string(root.RootDomain), libdnsRecords)
		case OverwriteDuplicate:
			libdnsRecords, err = provider.SetRecords(ctx, string(root.RootDomain), libdnsRecords)
		default:
			panic("unknown duplicate policy")
		}

		if err != nil {
			return fmt.Errorf("failed to apply records for %q: %w", root.RootDomain, err)
		}

		for _, record := range libdnsRecords {
			logger.Info(
				"applied libdns record",
				"record.type", record.Type,
				"record.name", record.Name,
				"record.value", record.Value)
		}

		return nil
	}

	var errs []error

	rootDomains, err := mapRootDomains(p)
	if err != nil {
		return err
	}

	// TODO: parallelize
	for _, root := range rootDomains {
		logger := logger.With(
			"provider", root.ProviderName,
			"root_domain", root.RootDomain)

		if err := apply(root, logger); err != nil {
			logger.Error(
				"cannot apply domain",
				"err", err)

			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
