package cfgutil

import (
	"encoding/json"
	"fmt"
)

// Discard is a special type that discards the JSON value.
type Discard struct{}

func (d *Discard) UnmarshalJSON(data []byte) error {
	return nil
}

// AssertSingleMapKey asserts that the JSON data is a single map key.
func AssertSingleMapKey(data []byte) error {
	var m map[string]Discard
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to parse single map key: %w", err)
	}
	if len(m) != 1 {
		keys := MapKeys(m)
		return fmt.Errorf("expected a single map key, got %d: %q", len(keys), keys)
	}
	return nil
}

// KeyValuePair represents a key-value pair. It is structured as a JSON object
// but guarantees that it has exactly one key.
type KeyValuePair[K ~string, V any] struct {
	Key   K
	Value V
}

func (kv *KeyValuePair[K, V]) UnmarshalJSON(data []byte) error {
	var m map[K]V
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to parse key-value pair: %w", err)
	}
	if len(m) != 1 {
		return fmt.Errorf("expected a single key-value pair, got %d", len(m))
	}
	for k, v := range m {
		kv.Key = k
		kv.Value = v
	}
	return nil
}

// MapKeys returns the keys of a map.
func MapKeys[K ~string, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
