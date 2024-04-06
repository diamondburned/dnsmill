package dnsmill

import (
	"encoding/json"
	"fmt"
)

// Config configures the behavior of the DNS tool.
type Config struct {
	DuplicatePolicy DuplicatePolicy `json:"duplicatePolicy,omitempty"`
}

// DefaultConfig returns the default configuration for the DNS tool.
func DefaultConfig() Config {
	return Config{
		DuplicatePolicy: ErrorOnDuplicate,
	}
}

// DuplicatePolicy is the policy to apply when a duplicate DNS record is found.
type DuplicatePolicy string

const (
	// ErrorOnDuplicate returns an error if a duplicate DNS record is found.
	// It is the safest policy and is the default.
	ErrorOnDuplicate DuplicatePolicy = "error"
	// OverwriteDuplicate overwrites the existing DNS record with the new one.
	OverwriteDuplicate DuplicatePolicy = "overwrite"
)

var allDuplicatePolicies = []DuplicatePolicy{
	ErrorOnDuplicate,
	OverwriteDuplicate,
}

func (d *DuplicatePolicy) UnmarshalJSON(data []byte) error {
	var policy string
	if err := json.Unmarshal(data, &policy); err != nil {
		return fmt.Errorf("failed to parse DuplicatePolicy: %w", err)
	}
	for _, p := range allDuplicatePolicies {
		if p == DuplicatePolicy(policy) {
			*d = p
			return nil
		}
	}
	return fmt.Errorf("invalid DuplicatePolicy: %s", policy)
}
