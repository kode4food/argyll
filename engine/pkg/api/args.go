package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
)

type (
	// Args represents a map of named arguments passed to or from steps
	Args map[Name]any

	// Name is a string identifier for arguments and attributes
	Name string

	argPair struct {
		K string `json:"k"`
		V any    `json:"v"`
	}
)

var (
	ErrMarshalArgs = errors.New("failed to marshal args")
)

// Set creates a new Args with the specified name-value pair added
func (a Args) Set(name Name, value any) Args {
	if a == nil {
		return Args{name: value}
	}
	res := maps.Clone(a)
	res[name] = value
	return res
}

// GetString retrieves a string value from args, returning defaultValue if not
// found or wrong type
func (a Args) GetString(name Name, defaultValue string) string {
	val, ok := a[name]
	if !ok {
		return defaultValue
	}
	str, ok := val.(string)
	if !ok {
		return defaultValue
	}
	return str
}

// GetBool retrieves a boolean value from args, returning defaultValue if not
// found or wrong type
func (a Args) GetBool(name Name, defaultValue bool) bool {
	val, ok := a[name]
	if !ok {
		return defaultValue
	}
	b, ok := val.(bool)
	if !ok {
		return defaultValue
	}
	return b
}

// GetInt retrieves an integer value from args, returning defaultValue if not
// found or wrong type. Supports both int and float64 (converting from JSON
// numbers)
func (a Args) GetInt(name Name, defaultValue int) int {
	val, ok := a[name]
	if !ok {
		return defaultValue
	}
	if i, ok := val.(int); ok {
		return i
	}
	if f, ok := val.(float64); ok {
		return int(f)
	}
	return defaultValue
}

// HashKey computes a deterministic SHA256 hash key of the Args. Keys are
// sorted alphabetically to ensure consistent hashing regardless of map
// iteration order. Returns hex string (64 chars) for use as cache key
func (a Args) HashKey() (string, error) {
	if len(a) == 0 {
		return sha256Hex(""), nil
	}

	keys := make([]Name, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	pairs := make([]argPair, len(keys))
	for i, k := range keys {
		pairs[i] = argPair{K: string(k), V: a[k]}
	}

	data, err := json.Marshal(pairs)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrMarshalArgs, err)
	}

	return sha256Hex(string(data)), nil
}

func sha256Hex(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}
